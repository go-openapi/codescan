// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package sentinel

type SentinelError string

func (e SentinelError) Error() string {
	return string(e)
}

// What the command refuses, and what it reports about a document it did produce.
//
// Sentinel values are used to decide the exit status, by asking which kind of refusal was met,
// so that a shell can tell "the scan failed" from "the scan is fine and you asked me to be strict about it".
const (
	// ErrUsage is a command line that does not make sense.
	ErrUsage SentinelError = "incorrect usage"

	// ErrDiagnostics is a run whose findings reached the severity -fail-on names.
	ErrDiagnostics SentinelError = "reported findings reached the -fail-on threshold"

	// ErrInvalidSpec is a document that -validate found invalid.
	ErrInvalidSpec SentinelError = "the specification is not valid"
)
