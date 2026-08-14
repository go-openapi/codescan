// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package humanize renders durations, sizes and counts the way the chrome shows them to a reader.
//
// Its own package because the header line and the run-cost overlay both need the same spellings, and neither owns the
// other: a duration that reads "1m 3s" in one place and "63s" in the other is the same measurement told two ways.
package humanize
