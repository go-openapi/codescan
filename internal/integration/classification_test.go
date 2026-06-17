// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"
)

// TestClassification_Spec is the full-pipeline snapshot of the canonical
// classification fixture (schemas + paths with their parameters, responses and
// operations), captured across the (SkipExtensions, DescWithRef) option matrix.
// It is the integration home for the snapshots that used to live as per-builder
// unit goldens (classification_{schema,params,responses,routes,operations}_*).
func TestClassification_Spec(t *testing.T) {
	cases := []struct {
		name    string
		skipExt bool
		descRef bool
		golden  string
	}{
		{"default", false, false, "classification_spec.json"},
		{"descwithref", false, true, "classification_spec_descwithref.json"},
		{"skipext", true, false, "classification_spec_skipext.json"},
		{"skipext_descwithref", true, true, "classification_spec_skipext_descwithref.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := codescan.Run(&codescan.Options{
				Packages: []string{
					"./goparsing/classification",
					"./goparsing/classification/models",
					"./goparsing/classification/operations",
				},
				WorkDir:        scantest.FixturesDir(),
				ScanModels:     true,
				SkipExtensions: tc.skipExt,
				DescWithRef:    tc.descRef,
			})
			require.NoError(t, err)
			require.NotNil(t, doc)
			scantest.CompareOrDumpJSON(t, doc, tc.golden)
		})
	}
}
