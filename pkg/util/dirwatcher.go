/*
   OqtaDrive - Sinclair Microdrive emulator
   Copyright (c) 2021, Alexander Vollschwitz

   This file is part of OqtaDrive.

   OqtaDrive is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   OqtaDrive is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with OqtaDrive. If not, see <http://www.gnu.org/licenses/>.
*/

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

/*
	NewDirWatcher creates a new recursive file system watcher that will watch
	for changes in the directory tree rooted in dir. When new directories are
	added to that tree, they will be included in the watch. The watcher will not
	start until the Start method has been called.
*/
func NewDirWatcher(dir string) (*DirWatcher, error) {

	ret := &DirWatcher{
		flush:   make(chan bool),
		release: make(chan bool),
	}

	var err error
	if ret.watcher, err = fsnotify.NewWatcher(); err != nil {
		return nil, err
	}

	if err := filepath.Walk(dir, ret.addDirWalking); err != nil {
		log.Errorf("error walking directory '%s': %v", dir, err)
		return nil, err
	}

	return ret, nil
}

//
type DirWatcher struct {
	watcher *fsnotify.Watcher
	flush   chan bool
	release chan bool
	running bool
}

/*
	Start starts this directory watcher. Whenever there is a change in the
	directory tree this watcher is watching, the handler function will be called.

	Additionally, a timer is set to expire after backoff time. If there were no
	further changes in the tree by the time the timer expires, the flush function
	will be called. Otherwise the timer is set again. The dir watcher is offering
	this service so that flushing is done from the same go routine, so the client
	does not have to be thread safe.
*/
func (dw *DirWatcher) Start(backoff time.Duration,
	handler func(fsnotify.Event) error, flush func() error) error {

	if dw.watcher == nil {
		return fmt.Errorf("directory watcher not initialized or stopped")
	}

	if dw.running {
		return fmt.Errorf("directory watcher already started")
	}

	dw.running = true

	go func() {

		timer := time.NewTimer(time.Millisecond)
		<-timer.C

		for {
			select {

			case evt, ok := <-dw.watcher.Events:

				if !ok {
					log.Debug("directory watcher routine stopping")
					dw.release <- true
					dw.running = false
					log.Debug("directory watcher routine exiting")
					return
				}

				timer.Stop()
				dw.handleEvent(evt)
				if err := handler(evt); err != nil {
					log.Errorf("error in watch event handler: %v", err)
				}
				timer = time.NewTimer(backoff)

			case err, ok := <-dw.watcher.Errors:
				if ok {
					log.Errorf("directory watcher error: %v", err)
				}

			case <-timer.C:
				if err := flush(); err != nil {
					log.Errorf("error flushing: %v", err)
				}
			}
		}
	}()

	return nil
}

/*
	Stop signals this directory watcher to stop, and waits until it has stopped.
	A stopped directory watcher cannot be started again. Returns immediately if
	this directory watcher is not running.
*/
func (dw *DirWatcher) Stop() {
	if dw.watcher != nil {
		log.Info("closing directory watcher")
		if err := dw.watcher.Close(); err != nil {
			log.Errorf("could not close file watcher: %v", err)
		}
		<-dw.release
		dw.watcher = nil
	}
}

//
func (dw *DirWatcher) handleEvent(evt fsnotify.Event) {
	log.WithFields(
		log.Fields{"path": evt.Name, "op": evt.Op}).Debug("handling event")
	if evt.Op == fsnotify.Create {
		dw.addDir(evt.Name)
	}
}

//
func (dw *DirWatcher) addDir(path string) error {
	return dw.handleDir(path, nil, nil, true)
}

//
func (dw *DirWatcher) addDirWalking(
	path string, info os.FileInfo, err error) error {
	return dw.handleDir(path, info, err, true)
}

//
func (dw *DirWatcher) removeDir(path string) error {
	return dw.handleDir(path, nil, nil, false)
}

//
func (dw *DirWatcher) handleDir(
	path string, info os.FileInfo, err error, add bool) error {

	if add && info == nil {
		var e error
		if info, e = os.Lstat(path); e != nil {
			log.Errorf("cannot stat %s", path)
			return e
		}
	}

	if add {
		if info.Mode().IsDir() {
			if err := dw.watcher.Add(path); err != nil {
				log.Errorf(
					"error adding watch for directory '%s': %v", path, err)
				return err
			}
			log.WithField("path", path).Debug("starting directory watch")

		}
	} else {
		if err := dw.watcher.Remove(path); err != nil {
			log.Errorf(
				"error removing watch for directory '%s': %v", path, err)
			return err
		}
		log.WithField("path", path).Debug("stopping directory watch")
	}

	return nil
}
