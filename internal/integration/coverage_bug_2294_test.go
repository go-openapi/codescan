// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2294 locks the resolution of go-swagger issue #2294 ("more than one security
// header using AND logic").
//
// Two schemes combined in ONE security requirement (a sequence item with two mapping keys) means
// AND — both must be satisfied — and must produce a single requirement object with both keys.
// Before the fix the scanner split it into two separate requirements (OR logic).
//
// The fix lives in the meta-Security parser (internal/parsers/security), the same parser as
// #2403/#2479.
func TestCoverage_Bug2294(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2294/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	require.Len(t, doc.Security, 1, "AND logic is a SINGLE requirement with two keys, not two requirements (go-swagger#2294)")
	req := doc.Security[0]
	_, hasClient := req["x_client_id"]
	_, hasToken := req["access_token"]
	assert.True(t, hasClient && hasToken, "both schemes share one requirement object (AND)")
}
