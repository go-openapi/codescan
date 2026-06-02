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
