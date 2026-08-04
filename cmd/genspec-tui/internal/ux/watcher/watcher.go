// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Watcher reports Go-source changes under a directory tree as a coalesced stream of signals.
//
// It watches every (non-vendored, non-hidden) directory in the tree — fsnotify is not recursive — and re-adds
// directories created at runtime so new packages are picked up.
// Bursts collapse into a single pending signal; the model debounces further before rescanning.
type Watcher struct {
	fs     *fsnotify.Watcher
	events chan struct{}
}

func New(root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{fs: fw, events: make(chan struct{}, 1)}
	w.addRecursive(root)
	go w.loop()
	return w, nil
}

// Close stops watching and tears down the goroutine.
func (w *Watcher) Close() error { return w.fs.Close() }

func (w *Watcher) Events() <-chan struct{} { return w.events }

// addRecursive adds every directory under root to the watch set, pruning the same noise the source tree prunes (hidden
// dirs, vendor, node_modules).
func (w *Watcher) addRecursive(root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil //nolint:nilerr // skip unreadable entries, keep walking
		}
		if name := d.Name(); path != root && (strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules") {
			return filepath.SkipDir
		}
		_ = w.fs.Add(path) // best effort; ignore per-dir watch-limit errors
		return nil
	})
}

func (w *Watcher) loop() {
	for {
		select {
		case ev, ok := <-w.fs.Events:
			if !ok {
				close(w.events)
				return
			}
			if w.relevant(ev) {
				w.signal()
			}
		case _, ok := <-w.fs.Errors:
			if !ok {
				close(w.events)
				return
			}
		}
	}
}

// relevant reports whether an event should trigger a rescan: any *.go change, or a directory create/remove/rename
// (which can add or drop packages).
//
// Newly created directories are added to the watch set so their files are seen.
func (w *Watcher) relevant(ev fsnotify.Event) bool {
	if strings.HasSuffix(ev.Name, ".go") {
		return true
	}
	if ev.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
		if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
			if ev.Op&fsnotify.Create != 0 {
				w.addRecursive(ev.Name)
			}
			return true
		}
	}
	return false
}

// signal posts a coalesced change notification (non-blocking: a pending signal already covers this change).
func (w *Watcher) signal() {
	select {
	case w.events <- struct{}{}:
	default:
	}
}
