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

// TestCoverage_Bug2520 locks go-swagger issue #2520 ("generate spec fails with unsupported type
// 'invalid type'"): a struct with a custom MarshalJSON and only unexported fields no longer aborts
// the scan — it emits an (empty) object and a $ref from the using field. codescan works at the
// type level and cannot infer the custom-marshaled wire shape; declaring it requires swagger:type /
// swagger:strfmt on the wrapper type.
func TestCoverage_Bug2520(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2520/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err, "a custom-marshaled / unexported-only struct must not abort the scan (go-swagger#2520)")

	name := doc.Definitions["Holder"].Properties["name"]
	assert.Equal(t, "#/definitions/String", name.Ref.String())
	s, ok := doc.Definitions["String"]
	require.True(t, ok)
	assert.Equal(t, []string{"object"}, []string(s.Type), "no custom-marshal inference: emits an object")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2520_schema.json")
}
