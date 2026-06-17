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

// TestCoverage_Bug2802 locks the fix for go-swagger issue #2802 ("Error
// generating spec if swagger:model is a generic struct"): an instantiated
// generic (`WrappedRequest[Request]`) used to panic with "can't determine
// refined type T (*types.TypeParam)".
//
// It no longer panics: the instantiated type resolves its type argument
// (`Body.Body -> $ref: Request`). The uninstantiated generic's free type
// parameter `T` is skipped with a warning (it has no spec representation),
// which is acceptable — the point is the scan completes.
func TestCoverage_Bug2802(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2802/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	inst, ok := doc.Definitions["WrappedRequestInstance"]
	require.True(t, ok)
	body, ok := inst.Properties["Body"]
	require.True(t, ok)
	inner, ok := body.Properties["Body"]
	require.True(t, ok)
	assert.Equal(t, "#/definitions/Request", inner.Ref.String(),
		"the instantiated type argument must resolve to a $ref")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2802_schema.json")
}
