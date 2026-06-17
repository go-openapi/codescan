// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestParsePatternPropertyPairs(t *testing.T) {
	cases := []struct {
		name   string
		arg    string
		want   []patternPropPair
		wantOK bool
	}{
		{
			name:   "single primitive",
			arg:    `"^x-": string`,
			want:   []patternPropPair{{regex: "^x-", spec: "string"}},
			wantOK: true,
		},
		{
			name: "multi pair with backslash and ref",
			arg:  `"^x-": string, "^\d+$": integer, "^item-": Item`,
			want: []patternPropPair{
				{regex: "^x-", spec: "string"},
				{regex: `^\d+$`, spec: "integer"},
				{regex: "^item-", spec: "Item"},
			},
			wantOK: true,
		},
		{
			name:   "regex containing a comma is not a delimiter",
			arg:    `"a{1,3}": string`,
			want:   []patternPropPair{{regex: "a{1,3}", spec: "string"}},
			wantOK: true,
		},
		{
			name:   "escaped quote inside regex",
			arg:    `"^\"x": string`,
			want:   []patternPropPair{{regex: `^"x`, spec: "string"}},
			wantOK: true,
		},
		{"missing quotes", `^x-: string`, nil, false},
		{"missing colon", `"^x-" string`, nil, false},
		{"empty spec", `"^x-": `, nil, false},
		{"unterminated regex", `"^x-: string`, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parsePatternPropertyPairs(tc.arg)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
