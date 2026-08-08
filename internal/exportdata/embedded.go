// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build exportdata

package exportdata

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"io/fs"
	"sync"
)

//go:embed exportdata.zip
var archive []byte

var (
	once   sync.Once
	reader *zip.Reader
)

// Embedded returns the embedded export data.
//
// A zip is used rather than a directory of embedded files because archive/zip's reader already
// implements fs.FS: one file in the binary, still read one package at a time.
func Embedded() (fs.FS, bool) {
	once.Do(func() {
		r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return // a corrupt archive degrades to "nothing embedded" rather than killing the scan
		}
		reader = r
	})
	if reader == nil {
		return nil, false
	}

	return reader, true
}
