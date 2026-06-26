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

// TestCoverage_Bug334 locks the resolution to go-swagger issue #334 ("not generating model info /
// empty definitions, Swagger UI error"): a swagger:route annotation placed INSIDE a function body
// is now detected, and the model is emitted — so definitions and paths are no longer empty.
//
// 📖 Need doc: when `// swagger:model` is placed BEFORE the doc text (annotation-first), the
// title/description are dropped — they are only captured when the annotation comes last.
// Asserted explicitly below so a future change is conscious.
func TestCoverage_Bug334(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/334/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	// The fix: definitions and paths are populated.
	user, ok := doc.Definitions["User"]
	require.True(t, ok, "the model is emitted (definitions not empty)")
	require.NotNil(t, doc.Paths)
	op := doc.Paths.Paths["/users"]
	require.NotNil(t, op.Get, "the in-function-body swagger:route is detected")
	assert.Equal(t, "users", op.Get.ID)
	require.Contains(t, doc.Responses, "usersResponse")

	// 📖 quirk: annotation-first ordering drops the doc text.
	assert.Empty(t, user.Title, "annotation-first ordering drops the title (doc point)")
	assert.Empty(t, user.Description, "annotation-first ordering drops the description")

	scantest.CompareOrDumpJSON(t, doc, "bugs_334_schema.json")
}
