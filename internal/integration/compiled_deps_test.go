// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// CompiledDependencies buys a large amount of time by taking dependency types from the compiler.
//
// It used to cost dependency source along with it, and this test pinned the loss. It no longer does:
// a dependency whose source carries annotations is parsed for them after the load, so the case that
// matters most here is the case that must NOT change.
//
// strfmt marks its own types — `// swagger:strfmt date-time` sits above strfmt.DateTime — and those
// marks are the only reason a field of that type acquires a format. Losing them was invisible in the
// output: a valid spec that quietly said less.
func TestCompiledDependencies_KeepsDependencyAnnotations(t *testing.T) {
	t.Parallel()

	scan := func(compiled bool) (format string, sourceless []string) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:             []string{"./petstore/..."},
			WorkDir:              scantest.FixturesDir() + "/goparsing",
			ScanModels:           true,
			CompiledDependencies: compiled,
			OnDiagnostic: func(d codescan.Diagnostic) {
				if strings.Contains(d.Message, "could not be read") {
					sourceless = append(sourceless, d.Message)
				}
			},
		})
		require.NoError(t, err)

		order, ok := doc.Definitions["order"]
		require.True(t, ok)
		orderedAt, ok := order.Properties["orderedAt"]
		require.True(t, ok)

		return orderedAt.Format, sourceless
	}

	format, sourceless := scan(false)
	assert.Equal(t, "date-time", format, "reading strfmt's source is what supplies the format")
	assert.Empty(t, sourceless, "every declaration a builder wanted was there to read")

	format, sourceless = scan(true)
	assert.Equal(t, "date-time", format,
		"strfmt carries the marker, so its source is read back and the format survives the compiled load")
	assert.NotContains(t, strings.Join(sourceless, "\n"), "strfmt",
		"and no builder went looking for a strfmt declaration and missed")

	// What DOES appear here is a lookup for time.Time, and it costs the spec nothing: the type is
	// recognized from its identity, so the declaration was never needed. It is announced because the
	// lookup happened, not because anything was lost — the recognizers that make it redundant run
	// after it at that call site rather than before. Asserted loosely for that reason: the notice is
	// honest about what the scanner did, and tightening it here would pin an ordering that is worth
	// changing.
	assert.NotEmpty(t, format)
}

// The complement — a dependency that says nothing about itself but declares a type the document carries — is
// TestDependencyDeclarations_ReadBackOnDemand, which is where the on-demand fetch and the field bridge are pinned.
