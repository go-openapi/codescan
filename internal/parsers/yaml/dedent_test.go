// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// TestRemoveIndent_LeadingBlankLine pins quirk F7: when a body starts with a blank line (the one
// gofmt inserts under a column-0 doc-comment key), the dedent strip width must come from the first
// non-blank line, not the blank line — otherwise the tab-indented children survive and the YAML
// parser rejects them.
//
// The fixed output dedents and retabs identically to the blank-line-free form.
func TestRemoveIndent_LeadingBlankLine(t *testing.T) {
	// gofmt-canonical shape: blank line, key at one tab, children at two.
	withBlank := []string{"", "\toauth2:", "\t\ttype: oauth2", "\t\tin: header"}
	// Same content without the leading blank.
	noBlank := []string{"\toauth2:", "\t\ttype: oauth2", "\t\tin: header"}

	got := RemoveIndent(withBlank)
	want := []string{"", "oauth2:", "  type: oauth2", "  in: header"}
	assert.Equal(t, want, got)

	// The blank-free form dedents to the same content (minus the blank).
	assert.Equal(t, want[1:], RemoveIndent(noBlank))

	// No tabs survive in the leading whitespace of any line.
	for _, l := range got {
		lead := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		assert.NotContains(t, lead, "\t", "leading tab must be retabbed: %q", l)
	}
}

// TestRemoveIndent_InterleavedProseAndCodeBlocks pins the swagger:operation counterpart of F7: a
// gofmt-canonical operation body interleaves prose-rendered top-level keys (one leading space) with
// tab-prefixed value blocks.
//
// Expanding leading tabs to spaces BEFORE the first-non-blank-line strip preserves the relative
// nesting; the previous strip-then-retab approach stripped the children's lone tab off and
// flattened them.
func TestRemoveIndent_InterleavedProseAndCodeBlocks(t *testing.T) {
	in := []string{
		" responses:",
		"",
		"\t'200':",
		"\t  description: ok",
		"",
		" x-ext:",
		"",
		"\thttpMethod: GET",
	}
	want := []string{
		"responses:",
		"",
		" '200':",
		"   description: ok",
		"",
		"x-ext:",
		"",
		" httpMethod: GET",
	}
	got := RemoveIndent(in)
	assert.Equal(t, want, got)

	// No tab survives in any line's leading whitespace.
	for _, l := range got {
		lead := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		assert.NotContains(t, lead, "\t", "leading tab must be expanded: %q", l)
	}
}

// TestRemoveIndent_ColumnZeroKey checks that a column-0 first non-blank line (no indent to strip)
// leaves the body untouched.
func TestRemoveIndent_ColumnZeroKey(t *testing.T) {
	in := []string{"", "oauth2:", "  type: oauth2"}
	assert.Equal(t, in, RemoveIndent(in))
}

// TestRemoveIndent_AllBlank returns the input unchanged when every line is blank (no canonical line
// to key off).
func TestRemoveIndent_AllBlank(t *testing.T) {
	in := []string{"", "  ", "\t"}
	assert.Equal(t, in, RemoveIndent(in))
}
