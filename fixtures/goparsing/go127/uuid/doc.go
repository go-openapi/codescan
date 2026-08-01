// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package uuidmodels witnesses the identity-based recognizer for the go1.27 stdlib uuid package.
//
// This file is deliberately UNTAGGED while model.go carries `//go:build go1.27`: under an older
// toolchain `import "uuid"` does not compile, and a package left with no buildable files at all
// fails `go build ./...` with "build constraints exclude all Go files". This doc file keeps the
// package loadable while contributing nothing to the emitted spec.
package uuidmodels
