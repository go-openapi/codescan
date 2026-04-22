// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestJoinDropLast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"nil", nil, ""},
		{"empty slice", []string{}, ""},
		{"single line", []string{"hello"}, "hello"},
		{"trailing blank dropped", []string{"hello", "world", "  "}, "hello\nworld"},
		{"trailing empty dropped", []string{"hello", "world", ""}, "hello\nworld"},
		{"no trailing blank", []string{"hello", "world"}, "hello\nworld"},
		{"only blank", []string{" "}, ""},
		{"only empty", []string{""}, ""},
		{"middle blank preserved", []string{"hello", "", "world"}, "hello\n\nworld"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualT(t, tc.want, JoinDropLast(tc.lines))
		})
	}
}

func TestSetter(t *testing.T) {
	t.Parallel()

	var target string
	set := Setter(&target)
	set([]string{"line1", "line2", ""})
	assert.EqualT(t, "line1\nline2", target)
}
