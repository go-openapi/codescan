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

// CompiledDependencies buys a large amount of time and costs dependency source. This pins what that
// costs in the case that matters most here, so nobody rediscovers it in a published spec.
//
// strfmt marks its own types — `// swagger:strfmt date-time` sits above strfmt.DateTime — and those
// marks are the only reason a field of that type acquires a format. Export data carries the type and
// not the comment, so the format goes and the field stays.
func TestCompiledDependencies_LosesDependencyAnnotations(t *testing.T) {
	t.Parallel()

	scan := func(compiled bool) (format string, hints int) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:             []string{"./petstore/..."},
			WorkDir:              scantest.FixturesDir() + "/goparsing",
			ScanModels:           true,
			CompiledDependencies: compiled,
			OnDiagnostic: func(d codescan.Diagnostic) {
				if d.Code == "scan.compiled-dependencies" {
					hints++
				}
			},
		})
		require.NoError(t, err)

		order, ok := doc.Definitions["order"]
		require.True(t, ok)
		orderedAt, ok := order.Properties["orderedAt"]
		require.True(t, ok)

		return orderedAt.Format, hints
	}

	format, hints := scan(false)
	assert.Equal(t, "date-time", format, "reading strfmt's source is what supplies the format")
	assert.Zero(t, hints)

	format, hints = scan(true)
	assert.Empty(t, format, "export data carries strfmt's types but not its annotations")
	assert.Equal(t, 1, hints, "and the loss is announced, since nothing in the output would show it")
}
