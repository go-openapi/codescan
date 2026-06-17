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

// TestCoverage_Bug1063 locks the resolution to go-swagger issue #1063 ("making
// a model readOnly"): a `// read only: true` annotation emits `readOnly: true`
// on the property.
//
// 📖 Need doc: readOnly is a spec-level marker; excluding the field from request
// bodies is the responsibility of downstream code generation / validation, not
// the spec scanner.
func TestCoverage_Bug1063(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1063/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	user := doc.Definitions["User"]
	assert.True(t, user.Properties["companyID"].ReadOnly, "read only: true -> readOnly")
	assert.False(t, user.Properties["firstname"].ReadOnly)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1063_schema.json")
}
