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
	"fmt"
	"os"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/lindb/common/pkg/fileutil"
	"github.com/lindb/common/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestWatch_Initialize(t *testing.T) {
	cases := []struct {
		name    string
		dir     string
		prepare func()
		wantErr bool
	}{
		{
			name: "create watcher failure",
			dir:  "./",
			prepare: func() {
				newWatcherFn = func() (*fsnotify.Watcher, error) {
					return nil, fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "mkdir failure",
			dir:  "./",
			prepare: func() {
				newWatcherFn = func() (*fsnotify.Watcher, error) {
					return nil, nil
				}
				mkdirIfNotExistFn = func(path string) error {
					return fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "add watch failure",
			dir:  "./test-watch",
			prepare: func() {
				mkdirIfNotExistFn = func(path string) error {
					return nil
				}
			},
			wantErr: true,
		},
		{
			name: "Initialize succes",
			dir:  "./",
			prepare: func() {
				mkdirIfNotExistFn = func(path string) error {
					return nil
				}
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				newWatcherFn = fsnotify.NewWatcher
				mkdirIfNotExistFn = fileutil.MkDirIfNotExist
			}()
			w := NewWatch(tt.dir, func(_ string, _ Op) {})
			tt.prepare()
			err := w.Initialize()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestWatch_Run(t *testing.T) {
	w := NewWatch("./", func(_ string, _ Op) {})
	assert.NoError(t, w.Initialize())
	w1 := w.(*watch)
	w.Run()
	c := make(chan struct{})
	go func() {
		w1.watcher.Errors <- fmt.Errorf("err")
		close(w1.watcher.Errors)
		c <- struct{}{}
	}()
	<-c

	w = NewWatch("./", func(_ string, _ Op) {})
	assert.NoError(t, w.Initialize())
	w1 = w.(*watch)
	w.Run()
	c = make(chan struct{})
	go func() {
		w1.watcher.Events <- fsnotify.Event{}
		close(w1.watcher.Events)
		c <- struct{}{}
	}()
	<-c
}

func TestWatch_Shutdown(t *testing.T) {
	w := NewWatch(".", func(fileName string, op Op) {})
	assert.Nil(t, w.Initialize())
	assert.Nil(t, w.Shutdown())
	w1 := w.(*watch)
	assert.Nil(t, w1.walkDir("path", nil, fmt.Errorf("err")))
	w1.watch("./test-watch") // watch failure

	w = &watch{}
	assert.Nil(t, w.Shutdown())
}

func TestWatch_handleEvent(t *testing.T) {
	ckFn := func(_ string, _ Op) {}
	w := &watch{
		notifyFn: ckFn,
		logger:   logger.GetLogger("Test", "Watch"),
	}
	cases := []struct {
		name    string
		prepare func()
		event   fsnotify.Event
	}{
		{
			name:  "panic",
			event: fsnotify.Event{Op: fsnotify.Remove},
			prepare: func() {
				w.notifyFn = func(_ string, _ Op) {
					panic("err")
				}
			},
		},
		{
			name:  "temp file",
			event: fsnotify.Event{Name: "1.txt~"},
		},
		{
			name:  "stat file failure",
			event: fsnotify.Event{Name: "1.txt", Op: fsnotify.Chmod},
			prepare: func() {
				statFn = func(name string) (os.FileInfo, error) {
					return nil, fmt.Errorf("err")
				}
			},
		},
		{
			name:  "watch sub dir",
			event: fsnotify.Event{Name: "../", Op: fsnotify.Chmod},
		},
		{
			name:  "file change",
			event: fsnotify.Event{Name: "./watch.go", Op: fsnotify.Chmod},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			defer func() {
				statFn = os.Stat
				w.notifyFn = ckFn
			}()
			if tt.prepare != nil {
				tt.prepare()
			}
			w.handleEvent(tt.event)
		})
	}
}
