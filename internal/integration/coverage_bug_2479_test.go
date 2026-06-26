// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2479 locks the resolution of go-swagger issue #2479 ("how to disable security on
// a route but keep on all endpoints").
//
// A route with `Security: []` must emit an explicit empty `security: []` on the operation — the
// OpenAPI 2.0 idiom for opting OUT of global security.
// Before the fix the scanner dropped it entirely (no `security` key), so the operation silently
// inherited global security instead of overriding it.
//
// The fix makes security.Parse return a non-nil empty list for the explicit `[]` form, which
// go-openapi/spec marshals as `"security": []` rather than omitting it.
func TestCoverage_Bug2479(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2479/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/users"].Post
	require.NotNil(t, op)
	require.NotNil(t, op.Security, "Security: [] must emit an explicit (non-nil) empty security (go-swagger#2479)")
	assert.Empty(t, op.Security, "the override is an empty requirement list")

	// And it must survive marshaling as `"security": []`.
	b, err := json.Marshal(op)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(b), `"security":[]`),
		"the empty security array must be emitted, not omitted")
}
