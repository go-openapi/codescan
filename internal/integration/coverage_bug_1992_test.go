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

// TestCoverage_Bug1992 documents the answer to go-swagger issue #1992 ("hide
// parts of composition for a struct"): codescan emits one schema per Go type, so
// per-operation field hiding is out of scope, but the OAS2 idiom for
// server-assigned fields — `read only: true` → `readOnly: true` — is supported
// and is the recommended mechanism.
func TestCoverage_Bug1992(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1992/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["User"].Properties
	assert.True(t, props["id"].ReadOnly, "server-assigned id is readOnly")
	assert.True(t, props["created"].ReadOnly, "server-assigned created is readOnly")
	assert.False(t, props["name"].ReadOnly, "client-supplied name is not readOnly")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1992_schema.json")
}
