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
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/oqtadrive/pkg/util"
)

//
const replaceChars = "`~!@#$%^&*_-+=()[]{}|;:',.<>?"

var nameCleaner *strings.Replacer

//
func init() {
	rep := make([]string, 2*len(replaceChars))
	for ix, c := range replaceChars {
		rep[ix*2] = string(c)
		rep[ix*2+1] = " "
	}
	nameCleaner = strings.NewReplacer(rep...)
}

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

	var err error
	i := &Index{}

	if i.base, err = filepath.Abs(base); err != nil {
		return nil, err
	}
	if i.repo, err = filepath.Abs(repo); err != nil {
		return nil, err
	}

	logger := log.WithFields(log.Fields{"base": i.base, "repo": i.repo})

	if _, err := os.Stat(i.base); err != nil {
		if os.IsNotExist(err) {
			logger.Info("creating new index")
			i.index, err = bleve.New(i.base, bleve.NewIndexMapping())
		}

		if err != nil {
			logger.Errorf("cannot create index: %v", err)
			return nil, err
		}

		logger.Info("new index created")
		i.empty = true

	} else {
		logger.Info("opening index")
		i.index, err = bleve.Open(i.base)
		if err != nil {
			logger.Errorf("cannot open index: %v", err)
			return nil, err
		}

		logger.Info("index opened")
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
	log.Info("pruning index")
	if err := i.prune(); err != nil {
		return fmt.Errorf("error pruning index: %v", err)
	}
	log.WithField(
		"duration", time.Now().Sub(start)).Info("index pruning finished")

	start = time.Now()
	log.Info("updating index")
	if err := i.update(); err != nil {
		return fmt.Errorf("error updating index: %v", err)
	}
	log.WithField(
		"duration", time.Now().Sub(start)).Info("index update finished")

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
			i.removeEntry(id)
		}
	}
}

//
func (i *Index) update() error {

	var lastMod time.Time
	if !i.empty {
		if store, err := os.Stat(filepath.Join(i.base, "store")); err == nil {
			lastMod = store.ModTime()
			log.Debugf("last index mod time: %v", lastMod)
		}
	}

	i.empty = false

	return filepath.Walk(i.repo,

		func(path string, info os.FileInfo, err error) error {

			if i.stopped {
				return fmt.Errorf("forced exit")
			}

			if !info.IsDir() && info.ModTime().After(lastMod) {
				i.addEntry(i.makeRelative(path))
			}

			return nil
		})
}

//
func (i *Index) startWatching() error {
	log.Info("starting index repo watcher")
	var err error
	if i.watcher, err = util.NewDirWatcher(i.repo); err != nil {
		return err
	}
	return i.watcher.Start(5*time.Second, i.watchEvent, i.flushEvent)
}

//
func (i *Index) watchEvent(evt fsnotify.Event) error {

	rel := i.makeRelative(evt.Name)
	log.WithFields(log.Fields{"path": rel, "op": evt.Op}).Debug("index update")

	switch evt.Op {

	case fsnotify.Create:
		if info, err := os.Stat(evt.Name); err != nil {
			log.Errorf("cannot add new entry: %v", err)
		} else if !info.IsDir() {
			i.addEntry(rel)
		}

	case fsnotify.Rename:
		fallthrough
	case fsnotify.Remove:
		i.removeEntry(rel)

	default:
		log.Debug("no index update required")
	}

	return nil
}

//
func (i *Index) flushEvent() error {
	return i.batched(true)
}

//
func (i *Index) addEntry(path string) error {

	logger := log.WithField("file", path)
	logger.Debug("adding new entry to index")

	if err := i.batch.Index(
		path, Entry{Name: nameCleaner.Replace(path)}); err != nil {
		logger.Errorf("failed to batch entry add: %v", err)
		return err
	}

	return i.batched(false)
}

//
func (i *Index) removeEntry(path string) error {
	log.WithField("file", path).Debug("removing deleted entry from index")
	i.batch.Delete(path)
	return i.batched(false)
}

// This is not thread safe. However, after setting up an index instance, add and
// remove are only ever called from the dir watcher, no concurrency.
func (i *Index) batched(flush bool) error {

	if i.batchCount++; flush || i.batchCount > 100 {
		log.Debug("flushing pending index actions")
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
func (i *Index) Search(term string, max int) ([]string, error) {

	term = strings.TrimSpace(term)
	if term == "" {
		return nil, fmt.Errorf("no search term")
	}

	log.Debugf("searching for '%s'", term)
	query := bleve.NewQueryStringQuery(term)
	search := bleve.NewSearchRequestOptions(query, max, 0, false)
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

//
func (i *Index) makeRelative(path string) string {
	if len(path) > len(i.repo) && strings.HasPrefix(path, i.repo) {
		return path[len(i.repo)+1:]
	}
	return path
}
