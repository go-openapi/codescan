package grammar

import (
	"iter"
	"slices"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestStripBlockContinuation(t *testing.T) {
	for testCase := range stripBlockTestCases() {
		t.Run(testCase.Name, func(t *testing.T) {
			assert.EqualT(t, testCase.Expected, stripBlockContinuation(testCase.Input))
		})
	}
}

// TestNormalizeBullet covers the markdown-bullet → canonical `- ` rewrite that
// makes `* item` / `+ item` lists identifiable like `- item` (go-swagger#1726).
func TestNormalizeBullet(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"asterisk bullet", "* item", "- item"},
		{"plus bullet", "+ item", "- item"},
		{"dash bullet untouched", "- item", "- item"},
		{"plain prose untouched", "item", "item"},
		{"emphasis not a bullet", "*emphasis*", "*emphasis*"},
		{"bold not a bullet", "**bold**", "**bold**"},
		{"lone asterisk not a bullet", "*", "*"},
		{"asterisk then tab is not a CommonMark bullet", "*\titem", "*\titem"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualT(t, tc.want, normalizeBullet(tc.in))
		})
	}
}

// TestTrimContentPrefixBullets locks the end-to-end content-prefix behaviour:
// leading godoc decoration is shed and a markdown bullet is normalised, while
// the YAML fence and prose are preserved.
func TestTrimContentPrefixBullets(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"indented asterisk bullet", "  * fast", "- fast"},
		{"indented plus bullet", "  + up", "- up"},
		{"indented dash bullet", "  - red", "- red"},
		{"yaml fence preserved", "---", "---"},
		{"slashes shed", "/ note", "note"},
		{"table pipe shed", "| cell", "cell"},
		{"plain prose", "hello world", "hello world"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualT(t, tc.want, trimContentPrefix(tc.in))
		})
	}
}

type stripBlockTestCase struct {
	Name     string
	Input    string
	Expected string
}

func stripBlockTestCases() iter.Seq[stripBlockTestCase] {
	const blanks = " \t \u00a0 \u205f"

	return slices.Values([]stripBlockTestCase{
		{
			Name:     "empty string",
			Input:    "",
			Expected: "",
		},
		{
			Name:     "blank string with unicode whitespace",
			Input:    blanks,
			Expected: blanks,
		},
		{
			Name:     "all-whitespace returns input verbatim",
			Input:    "   ",
			Expected: "   ",
		},
		{
			Name:     "blanks with * string",
			Input:    blanks + "*",
			Expected: "",
		},
		{
			Name:     "blanks with * + indented string",
			Input:    blanks + "*\u2029  ",
			Expected: "  ",
		},
		{
			Name:     "blanks with * + indented string",
			Input:    blanks + "*\u2029  indented",
			Expected: "  indented",
		},
		{
			Name:     "blanks with * + string",
			Input:    blanks + "*notindented",
			Expected: "notindented",
		},
		{
			Name:     "no marker",
			Input:    "x",
			Expected: "x",
		},
		{
			Name:     "indented no marker",
			Input:    "  x",
			Expected: "  x",
		},
		{
			Name:     "canonical godoc continuation",
			Input:    " * hello",
			Expected: "hello",
		},
		{
			Name:     "Unicode whitespace around the marker",
			Input:    " * hello",
			Expected: "hello",
		},
		{
			Name:     "no marker preserves indentation",
			Input:    "  not_a_continuation",
			Expected: "  not_a_continuation",
		},
		{
			Name:     "marker without surrounding whitespace",
			Input:    "*hello",
			Expected: "hello",
		},
	})
}
