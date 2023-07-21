// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package fileutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/lindb/common/pkg/fileutil"
	"github.com/lindb/common/pkg/logger"
)

//go:generate mockgen -source=./watch.go -destination=./watch_mock.go -package=fileutil

// for testing
var (
	newWatcherFn      = fsnotify.NewWatcher
	mkdirIfNotExistFn = fileutil.MkDirIfNotExist
	statFn            = os.Stat
)

// Op represents file operation type.
type Op int

const (
	Modify Op = iota + 1
	Remove
)

// Watch represents file change event under dir and all sub dirs.
type Watch interface {
	// Initialize Initializes the file watch, and reads file init data.
	Initialize() error
	// Run runs file watch, then handles file change event.
	Run()
	// Shutdown Shutdowns file watch.
	Shutdown() error
}

// watch implements Watch interface.
type watch struct {
	dir      string
	watcher  *fsnotify.Watcher
	notifyFn func(fileName string, op Op)

	logger logger.Logger
}

// NewWatch creates a file Watch instance.
func NewWatch(dir string, nodifyFn func(fileName string, op Op)) Watch {
	return &watch{
		dir:      dir,
		notifyFn: nodifyFn,
		logger:   logger.GetLogger("FileUtil", "Watch"),
	}
}

// Initialize Initializes the file watch, and reads file init data.
func (w *watch) Initialize() error {
	watcher, err := newWatcherFn()
	if err != nil {
		return err
	}
	w.watcher = watcher
	if err := mkdirIfNotExistFn(w.dir); err != nil {
		return err
	}
	if err := w.watcher.Add(w.dir); err != nil {
		return err
	}
	return filepath.WalkDir(w.dir, w.walkDir)
}

// Run runs file watch, then handles file change event.
func (w *watch) Run() {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				w.logger.Warn("watch file change error", logger.Error(err))
			}
		}
	}()
}

// Shutdown Shutdowns file watch.
func (w *watch) Shutdown() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// handleEvent handles file change event.
func (w *watch) handleEvent(event fsnotify.Event) {
	fileName := event.Name
	defer func() {
		if err := recover(); err != nil {
			w.logger.Warn("handle file change event failure", logger.String("file", fileName), logger.Any("err", err))
		}
	}()
	if strings.HasSuffix(fileName, "~") {
		return
	}

	w.logger.Info("found file changed", logger.String("file", fileName), logger.String("op", event.Op.String()))
	switch event.Op {
	case fsnotify.Chmod, fsnotify.Create, fsnotify.Write:
		f, err := statFn(fileName)
		if err != nil {
			w.logger.Warn("get file info failure", logger.String("file", fileName), logger.Error(err))
			return
		}
		if f.IsDir() {
			w.watch(fileName)
			return
		}
		w.notifyFn(fileName, Modify)
	case fsnotify.Remove:
		w.notifyFn(fileName, Remove)
	}
}

func (w *watch) walkDir(path string, d fs.DirEntry, err error) error {
	if err != nil {
		w.logger.Warn("walk dir failure", logger.String("path", path), logger.Error(err))
		return nil
	}
	if d.IsDir() {
		return w.watcher.Add(path)
	}
	w.notifyFn(path, Modify)
	return nil
}

func (w *watch) watch(fileName string) {
	if err0 := w.watcher.Add(fileName); err0 != nil {
		w.logger.Warn("watch sub dir failure", logger.String("file", fileName), logger.Error(err0))
	}
}
