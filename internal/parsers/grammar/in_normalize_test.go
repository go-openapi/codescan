// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

func TestNormalizeIn(t *testing.T) {
	cases := []struct {
		raw     string
		alias   bool
		want    string
		wantOK  bool
		comment string
	}{
		{"body", false, "body", true, "canonical lowercase"},
		{"Body", false, "body", true, "Q29 — capitalised must canonicalise"},
		{"BODY", false, "body", true, "all-caps"},
		{"bODy", false, "body", true, "mixed-case"},
		{"query", false, "query", true, ""},
		{"QUERY", false, "query", true, ""},
		{"path", false, "path", true, ""},
		{"Path", false, "path", true, ""},
		{"header", false, "header", true, ""},
		{"Header", false, "header", true, ""},
		{"formData", false, "formData", true, "canonical camelCase"},
		{"formdata", false, "formData", true, "all-lower → canonical"},
		{"FORMDATA", false, "formData", true, "all-caps → canonical"},
		{"FormData", false, "formData", true, "title-case → canonical"},
		{"form", false, "", false, "Q27 — form alias OFF outside routebody"},
		{"Form", false, "", false, "form alias OFF case-insensitive"},
		{"form", true, "formData", true, "Q27 — form alias ON in routebody"},
		{"FORM", true, "formData", true, "form alias ON case-insensitive"},
		{"  body  ", false, "body", true, "leading/trailing whitespace trimmed"},
		{"", false, "", false, "empty"},
		{"  ", false, "", false, "whitespace-only"},
		{"cookie", false, "", false, "unknown vocab"},
		{"bodyish", false, "", false, "near-miss must not match"},
	}
	for _, tc := range cases {
		name := tc.raw + "/alias=" + boolStr(tc.alias)
		t.Run(name, func(t *testing.T) {
			got, ok := grammar.NormalizeIn(tc.raw, tc.alias)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("NormalizeIn(%q, %v) = (%q, %v); want (%q, %v) — %s",
					tc.raw, tc.alias, got, ok, tc.want, tc.wantOK, tc.comment)
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
