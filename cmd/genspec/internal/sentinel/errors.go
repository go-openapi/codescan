// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package sentinel

// Error is a constant error this command fails on: comparable, so errors.Is answers on identity, and
// declarable in a const block, so no package variable is mutable in the meantime.
type Error string

func (e Error) Error() string {
	return string(e)
}

// What the command refuses, and what it reports about a document it did produce.
//
// Sentinel values are used to decide the exit status, by asking which kind of refusal was met,
// so that a shell can tell "the scan failed" from "the scan is fine and you asked me to be strict about it".
const (
	// ErrUsage is a command line that does not make sense.
	ErrUsage Error = "incorrect usage"

	// ErrDiagnostics is a run whose findings reached the severity -fail-on names.
	ErrDiagnostics Error = "reported findings reached the -fail-on threshold"

	// ErrInvalidSpec is a document that -validate found invalid.
	ErrInvalidSpec Error = "the specification is not valid"
)
