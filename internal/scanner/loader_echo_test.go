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
func TestLoaderInUse(t *testing.T) {
	t.Log(testloader.Describe("scanner"))
}
