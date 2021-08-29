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

package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/util"
)

/*
	FIXME:
	- review log levels
*/

//
func NewIndex(base, repo string) (*Index, error) {
	if ret, err := createOrOpen(base, repo); err != nil {
		return nil, err
	} else {
		return ret, nil
	}
}

//
func createOrOpen(base, repo string) (*Index, error) {

	i := &Index{base: base, repo: repo}
	logger := log.WithField("index", i.base)

	if _, err := os.Stat(i.base); err != nil {
		if os.IsNotExist(err) {
			logger.Info("creating repo index")
			i.index, err = bleve.New(i.base, bleve.NewIndexMapping())
		}

		if err != nil {
			logger.Errorf("cannot create repo index: %v", err)
			return nil, err
		}

		logger.Info("repo index created")
		i.empty = true

	} else {
		logger.Info("opening repo index")
		i.index, err = bleve.Open(i.base)
		if err != nil {
			logger.Errorf("cannot open repo index: %v", err)
			return nil, err
		}

		logger.Info("repo index opened")
	}

	i.batch = i.index.NewBatch()
	return i, nil
}

//
type Entry struct {
	Name string
}

//
type Index struct {
	base    string
	repo    string
	stopped bool
	//
	index   bleve.Index
	empty   bool
	watcher *util.DirWatcher
	//
	batch      *bleve.Batch
	batchCount int
}

//
func (i *Index) Start() error {

	start := time.Now()
	if err := i.prune(); err != nil {
		return fmt.Errorf("error pruning index: %v", err)
	}
	log.Infof("pruning took %v", time.Now().Sub(start))

	start = time.Now()
	if err := i.update(); err != nil {
		return fmt.Errorf("error updating index: %v", err)
	}
	log.Infof("update took %v", time.Now().Sub(start))

	if err := i.startWatching(); err != nil {
		return fmt.Errorf("error starting repo watcher: %v", err)
	}

	if err := i.batched(true); err != nil {
		return err
	}

	log.Info("index ready")
	return nil
}

//
func (i *Index) Stop() {

	if i.watcher != nil {
		i.watcher.Stop()
	}

	if i.index != nil {
		i.index.Close()
	}

	i.stopped = true
}

//
func (i *Index) prune() error {

	if i.empty {
		return nil
	}

	logger := log.WithField("index", i.base)
	logger.Info("pruning deleted index entries")

	ix, err := i.index.Advanced()
	if err != nil {
		return err
	}

	rd, err := ix.Reader()
	if err != nil {
		return err
	}
	defer rd.Close()

	docs, err := rd.DocIDReaderAll()
	if err != nil {
		return err
	}
	defer docs.Close()

	for {
		d, err := docs.Next()
		if err != nil {
			return err
		}
		if d == nil {
			return nil
		}
		id, err := rd.ExternalID(d)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(i.repo, id)); os.IsNotExist(err) {
			i.removeEntry(id, true)
		}
	}
}

//
func (i *Index) update() error {

	log.WithField("index", i.base).Info("updating index")

	var lastMod time.Time
	if !i.empty {
		if store, err := os.Stat(filepath.Join(i.base, "store")); err == nil {
			lastMod = store.ModTime()
			log.Infof("last mod time: %v", lastMod)
		}
	}

	i.empty = false

	return filepath.Walk(i.repo,

		func(path string, info os.FileInfo, err error) error {

			if i.stopped {
				return fmt.Errorf("forced exit")
			}

			if !info.IsDir() && info.ModTime().After(lastMod) {
				i.addEntry(path[len(i.repo):], true)
			}

			return nil
		})
}

//
func (i *Index) startWatching() error {
	log.WithField("index", i.base).Info("starting repo watcher")
	var err error
	i.watcher, err = util.NewDirWatcher(i.repo)
	if err != nil {
		return err
	}
	return i.watcher.Start(5*time.Second, i.watchEvent, i.flushEvent)
}

//
func (i *Index) watchEvent(evt fsnotify.Event) error {

	rel := evt.Name[len(i.repo):]
	logger := log.WithFields(log.Fields{
		"path": rel,
		"op":   evt.Op,
	})
	logger.Info("repo update")

	switch evt.Op {

	case fsnotify.Create:
		if info, err := os.Stat(evt.Name); err != nil {
			logger.Errorf("cannot add new entry: %v", err)
		} else if !info.IsDir() {
			i.addEntry(rel, false)
		}

	case fsnotify.Rename:
		fallthrough
	case fsnotify.Remove:
		i.removeEntry(rel, false)

	default:
		logger.Info("no action required")
	}

	return nil
}

//
func (i *Index) flushEvent() error {
	log.Debug("flushing pending index actions")
	return i.batched(true)
}

//
func (i *Index) addEntry(path string, quiet bool) error {

	logger := log.WithField("file", path)
	if !quiet {
		logger.Info("adding new entry to index")
	}

	if err := i.batch.Index(path, Entry{Name: path}); err != nil {
		logger.Errorf("failed to batch entry add: %v", err)
		return err
	}

	return i.batched(false)
}

//
func (i *Index) removeEntry(path string, quiet bool) error {

	logger := log.WithField("file", path)
	if !quiet {
		logger.Info("removing deleted entry from index")
	}

	i.batch.Delete(path)

	return i.batched(false)
}

// This is not thread safe. However, after setting up an index instance, add and
// remove are only ever called from the dir watcher, no concurrency.
func (i *Index) batched(flush bool) error {

	if i.batchCount++; flush || i.batchCount > 100 {
		if err := i.index.Batch(i.batch); err != nil {
			log.Errorf("failed to execute index batch: %v", err)
			return err
		}
		i.batch = i.index.NewBatch()
		i.batchCount = 0
	}

	return nil
}

//
func (i *Index) Search(term string) ([]string, error) {

	log.Debugf("searching for '%s'", term)

	query := bleve.NewQueryStringQuery(term)
	search := bleve.NewSearchRequest(query)
	res, err := i.index.Search(search)
	if err != nil {
		return nil, err
	}

	ret := make([]string, len(res.Hits))
	for ix, h := range res.Hits {
		ret[ix] = h.ID
	}

	return ret, nil
}
