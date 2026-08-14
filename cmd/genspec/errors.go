// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"

	"github.com/go-openapi/codescan/internal/cliopts"
)

// What the command refuses, and what it reports about a document it did produce.
//
// Sentinels rather than a formatted string at each site: the exit status is decided by asking which
// kind of refusal was met, so a shell can tell "the scan failed" from "the scan is fine and you
// asked me to be strict about it" without matching on prose that is free to change.
var (
	// errUsage is a command line that does not make sense.
	errUsage = errors.New("bad usage")
	// errDiagnostics is a run whose findings reached the severity -fail-on names.
	errDiagnostics = errors.New("reported findings reached the -fail-on threshold")
	// errInvalidSpec is a document that -validate found invalid.
	errInvalidSpec = errors.New("the specification is not valid")
)

// Exit statuses. See the package documentation.
const (
	exitOK          = 0
	exitFailed      = 1
	exitUsage       = 2
	exitDiagnostics = 3
	exitInvalidSpec = 4
)

// exitStatus maps what run reported onto what the shell sees.
//
// Anything unrecognised is a failed scan: the specific statuses are promises about states the
// command distinguishes on purpose, and a new kind of error is not one of them until it is added
// here.
func exitStatus(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, errUsage), errors.Is(err, cliopts.ErrBadFlag):
		return exitUsage
	case errors.Is(err, errDiagnostics):
		return exitDiagnostics
	case errors.Is(err, errInvalidSpec):
		return exitInvalidSpec
	default:
		return exitFailed
	}
}
