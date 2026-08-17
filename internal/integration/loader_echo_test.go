// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/testloader"
)

// TestLoaderInUse records which loader the suite ran under.
//
// This package is the bulk of the run and the one whose cost moves most with that choice, so a run
// that comes in unexpectedly slow can be read rather than guessed at. `go test -json` captures a
// passing test's log, so the line survives in the report CI keeps even where the console hides it.
func TestLoaderInUse(t *testing.T) {
	t.Log(testloader.Describe("integration"))
}
