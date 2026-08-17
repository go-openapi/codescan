// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"os"
	"testing"

	"github.com/go-openapi/codescan/internal/testloader"
)

// TestMain says which loader the suite ran under.
//
// This package is the bulk of the run and the one whose cost moves most with that choice, so a run
// that is unexpectedly slow can be read rather than guessed at.
func TestMain(m *testing.M) {
	testloader.Announce("integration")
	os.Exit(m.Run())
}
