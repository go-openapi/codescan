// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package vfs

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FS is the loader's single point of filesystem contact.
//
// It exists so that every read — build-constraint matching inside go/build, directory walking during pattern
// resolution, and source reading before parsing — goes through one place that can be pointed at either the real
// filesystem or an fs.FS.
//
// It is its own package because both halves of the loader need it and neither owns it: resolution walks directories,
// and the loader reads the files resolution found.
type FS struct {
	fsys fs.FS // nil: real filesystem
}

// New returns an FS reading through fsys, or through the os package when fsys is nil.
func New(fsys fs.FS) *FS { return &FS{fsys: fsys} }

func (v *FS) Virtual() bool { return v.fsys != nil }

// Clean maps a caller-supplied path onto the convention of the active backend.
//
// io/fs requires slash-separated, unrooted paths, while callers naturally write OS paths and sometimes absolute ones.
// Normalising here (rather than rejecting) keeps the same patterns working against both backends.
func (v *FS) Clean(p string) string {
	if !v.Virtual() {
		return p
	}
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimLeft(p, "/")
	if p == "" {
		return "."
	}
	return path.Clean(p)
}

func (v *FS) Open(p string) (io.ReadCloser, error) {
	if !v.Virtual() {
		return os.Open(p)
	}
	return v.fsys.Open(v.Clean(p))
}

func (v *FS) ReadFile(p string) ([]byte, error) {
	if !v.Virtual() {
		return os.ReadFile(p)
	}
	return fs.ReadFile(v.fsys, v.Clean(p))
}

// ReadDir returns fs.FileInfo values because that is what go/build's ReadDir hook requires.
func (v *FS) ReadDir(dir string) ([]fs.FileInfo, error) {
	var entries []fs.DirEntry
	var err error
	if !v.Virtual() {
		entries, err = os.ReadDir(dir)
	} else {
		entries, err = fs.ReadDir(v.fsys, v.Clean(dir))
	}
	if err != nil {
		return nil, err
	}
	infos := make([]fs.FileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue // a vanished entry is not worth failing the whole directory over
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (v *FS) IsDir(p string) bool {
	if !v.Virtual() {
		info, err := os.Stat(p)
		return err == nil && info.IsDir()
	}
	info, err := fs.Stat(v.fsys, v.Clean(p))
	return err == nil && info.IsDir()
}

func (v *FS) Join(elem ...string) string {
	if !v.Virtual() {
		return filepath.Join(elem...)
	}
	return path.Join(elem...)
}

func (v *FS) IsAbs(p string) bool {
	if !v.Virtual() {
		return filepath.IsAbs(p)
	}
	return false // every path in an fs.FS is rooted at the FS itself
}

// HasSubdir reports whether dir is within root, and if so its relative path.
func (v *FS) HasSubdir(root, dir string) (string, bool) {
	if !v.Virtual() {
		rel, err := filepath.Rel(root, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", false
		}
		return filepath.ToSlash(rel), true
	}
	root, dir = v.Clean(root), v.Clean(dir)
	if root == "." {
		return dir, true
	}
	if dir == root {
		return ".", true
	}
	if !strings.HasPrefix(dir, root+"/") {
		return "", false
	}
	return strings.TrimPrefix(dir, root+"/"), true
}

// WalkDirs yields every directory at or below root, in lexical order.
func (v *FS) WalkDirs(root string, yield func(string) error) error {
	if !v.Virtual() {
		return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil //nolint:nilerr // an unreadable subtree is skipped, not fatal
			}
			if skipDir(d.Name()) && p != root {
				return filepath.SkipDir
			}
			return yield(p)
		})
	}
	// io/fs walks in its own rooted namespace, so every result has to be mapped back onto the form the caller uses.
	// Yielding the fs-internal path instead would leak into file names, and from there into every token.Position the scan
	// reports — a position an editor cannot resolve.
	inner := v.Clean(root)

	return fs.WalkDir(v.fsys, inner, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil //nolint:nilerr // as above
		}
		if skipDir(d.Name()) && p != inner {
			return fs.SkipDir
		}

		return yield(rebase(root, inner, p))
	})
}

// rebase maps a path produced by walking under inner back onto the caller's root.
func rebase(root, inner, p string) string {
	if p == inner {
		return root
	}

	rel := strings.TrimPrefix(p, inner+"/")
	if inner == "." {
		rel = p
	}

	return strings.TrimSuffix(root, "/") + "/" + rel
}

// skipDir reports directories the go tool itself never treats as packages.
//
// "vendor" is deliberately absent: it is not universally excluded, only unreachable THROUGH a wildcard, and a directory
// named vendor can be an ordinary package.
// That rule needs to know whether it is walking a wildcard, so the resolver applies it rather than this list.
func skipDir(name string) bool {
	return name == "testdata" ||
		strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
