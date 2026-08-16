// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import "errors"

// What a configuration file can be wrong about.
//
// Sentinels rather than a formatted string at each site, so that a command can decide what a bad
// configuration file costs it - a usage status, usually - without matching on prose.
var (
	// ErrBadConfig is a file that cannot be read or is not a mapping of sections.
	ErrBadConfig = errors.New("bad configuration file")
	// ErrUnknownKey is a key naming no flag of the command reading it.
	ErrUnknownKey = errors.New("unknown configuration key")
	// ErrBadValue is a value the flag it addresses will not take.
	ErrBadValue = errors.New("bad configuration value")
)
