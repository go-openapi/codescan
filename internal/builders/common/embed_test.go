// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestScanInLocation(t *testing.T) {
	cases := []struct {
		name      string
		text      string
		wantValue string
		wantValid bool
		wantInval string
	}{
		{"path", "in: path", "path", true, ""},
		{"query", "in: query", "query", true, ""},
		{"body", "in: body", "body", true, ""},
		{"capitalised key and value", "In: Body", "body", true, ""},
		{"trailing dot", "in: path.", "path", true, ""},
		{"amid prose", "Some doc.\nin: header\nmore.", "header", true, ""},
		{"absent", "just a description", "", false, ""},
		{"out of vocabulary", "in: sideband", "", false, "sideband"},
		{"form alias not accepted here", "in: form", "", false, "form"},
		{"empty value skipped", "in:\nin: path", "path", true, ""}, //nolint:dupword // intentional: an empty `in:` line followed by a valid one
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, ok, inval := ScanInLocation(tc.text)
			assert.Equal(t, tc.wantValue, v)
			assert.Equal(t, tc.wantValid, ok)
			assert.Equal(t, tc.wantInval, inval)
		})
	}
}
