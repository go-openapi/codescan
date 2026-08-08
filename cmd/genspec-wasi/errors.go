// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import "errors"

// What the command refuses before it starts.
//
// Sentinels rather than a formatted string at each site: a caller - a test, or a shell checking an
// exit - can then ask which kind of refusal it met without matching on prose that is free to change.
var (
	// errBadFlag is a flag whose value is not one of the ones it accepts.
	errBadFlag = errors.New("bad flag")
	// errBadExportData is a -export-data path that is neither a directory nor a .zip.
	errBadExportData = errors.New("unusable export data")
)
