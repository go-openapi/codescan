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

// TestCoverage_Bug2064 locks go-swagger issue #2064 ("add example to string
// parameter in request body"): the example and default of a body parameter are
// emitted (they were previously missing). They are carried on the parameter
// object alongside the body schema.
func TestCoverage_Bug2064(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2064/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/thing"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "body", p.In)
	assert.Equal(t, "example", p.Example, "the body param example is emitted (go-swagger#2064)")
	assert.Equal(t, "def", p.Default, "the body param default is emitted")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2064_schema.json")
}
