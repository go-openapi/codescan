// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestBlock_IsDeprecated(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		want       bool
		wantInPros string // substring that must survive in the prose ("" = skip)
	}{
		{
			name: "explicit deprecated true",
			src:  "Widget does things.\n\ndeprecated: true\n\nswagger:model Widget",
			want: true,
		},
		{
			name: "explicit deprecated false",
			src:  "Widget does things.\n\ndeprecated: false\n\nswagger:model Widget",
			want: false,
		},
		{
			name:       "godoc Deprecated paragraph",
			src:        "Widget does things.\n\nDeprecated: use NewWidget instead.\n\nswagger:model Widget",
			want:       true,
			wantInPros: "Deprecated: use NewWidget instead.",
		},
		{
			name: "no deprecation",
			src:  "Widget does things.\n\nswagger:model Widget",
			want: false,
		},
		{
			name: "Deprecated mid-sentence is not a mark",
			src:  "Widget is not Deprecated: it is fine.\n\nswagger:model Widget",
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := parseString(t, tc.src)
			assert.Equal(t, tc.want, b.IsDeprecated())

			// A godoc "Deprecated:" paragraph must never be mistaken for a
			// malformed bool keyword.
			for _, d := range b.Diagnostics() {
				assert.NotEqual(t, CodeInvalidBoolean, d.Code,
					"unexpected invalid-boolean diagnostic: %s", d)
			}

			if tc.wantInPros != "" {
				assert.True(t, strings.Contains(b.Prose(), tc.wantInPros),
					"prose %q must retain %q", b.Prose(), tc.wantInPros)
			}
		})
	}
}
