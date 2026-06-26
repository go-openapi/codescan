// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2917 locks the fix for go-swagger issue #2917 ("classifier: unknown swagger
// annotation \"extendee\""): a grpc-gateway generated comment containing
// `…openapiv2_swagger:extendee…` used to be matched by the over-broad annotation regex and
// aborted the scan.
//
// The current regex only matches `swagger:` at start-of-line or after whitespace/slash, so the
// embedded `_swagger:extendee` is not treated as an annotation.
func TestCoverage_Bug2917(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2917/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err, "an embedded `_swagger:extendee` comment must not abort the classifier")
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2917_schema.json")
}
