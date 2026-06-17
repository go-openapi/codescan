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

// TestCoverage_Bug2419 locks go-swagger issue #2419 ("support custom swagger type
// for struct field"): a `swagger:type string` on a field whose type is an
// external/imported struct overrides it to a plain string (instead of a $ref to
// the external type's definition).
func TestCoverage_Bug2419(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2419/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	uid := doc.Definitions["SetEmailRequest"].Properties["user_id"]
	assert.Equal(t, []string{"string"}, []string(uid.Type),
		"swagger:type overrides the external-type field to string (go-swagger#2419)")
	assert.Empty(t, uid.Ref.String(), "no $ref to the external type when overridden")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2419_schema.json")
}
