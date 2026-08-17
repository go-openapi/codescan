// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"testing"

	"github.com/go-openapi/codescan/internal/testloader"
)

// TestLoaderInUse records which loader the suite ran under.
//
// Worth having here in particular: this suite was silently left on the default while the setting was
// believed to cover it, and an echo would have said so at a glance.
//
// Skipped rather than logged, because gotestsum prints a skip's message and swallows a passing
// test's log. See the integration suite's twin.
func TestLoaderInUse(t *testing.T) {
	t.Skip(testloader.Describe("scanner"))
}
