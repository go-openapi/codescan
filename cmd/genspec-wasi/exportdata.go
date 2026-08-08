// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// openExportData resolves -export-data, which takes either a directory or a zip.
//
// The zip form exists for hosts that have somewhere to put a file but no directory tree to build: a
// browser drops one fetched blob into the guest filesystem instead of unpacking several hundred
// entries in JavaScript. archive/zip's reader is already an fs.FS, so nothing downstream can tell
// the difference.
func openExportData(path string) (fs.FS, error) {
	if !strings.HasSuffix(path, ".zip") {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("%w: %q is neither a directory nor a .zip", errBadExportData, path)
		}

		return os.DirFS(path), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening export data: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("opening export data: %w", err)
	}

	// The reader keeps the file open and reads entries on demand, which is the point: the archive is
	// several megabytes and a scan touches a fraction of it.
	r, err := zip.NewReader(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("reading export data archive: %w", err)
	}

	return r, nil
}
