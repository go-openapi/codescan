// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"errors"
	"fmt"
	"go/ast"
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/spec"
)

// only used within this group of tests but never used within actual code base.
func newSchemaAnnotationParser(goName string) *schemaAnnotationParser {
	return &schemaAnnotationParser{GoName: goName, rx: rxModelOverride}
}

type schemaAnnotationParser struct {
	GoName string
	Name   string
	rx     *regexp.Regexp
}

func (sap *schemaAnnotationParser) Matches(line string) bool {
	return sap.rx.MatchString(line)
}

func (sap *schemaAnnotationParser) Parse(lines []string) error {
	if sap.Name != "" {
		return nil
	}

	if len(lines) > 0 {
		for _, line := range lines {
			matches := sap.rx.FindStringSubmatch(line)
			if len(matches) > 1 && len(matches[1]) > 0 {
				sap.Name = matches[1]
				return nil
			}
		}
	}
	return nil
}

func TestSectionedParser_TitleDescription(t *testing.T) {
	const (
		text = `This has a title, separated by a whitespace line

In this example the punctuation for the title should not matter for swagger.
For go it will still make a difference though.
`
		text2 = `This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.
`

		text3 = `This has a title, and markdown in the description

See how markdown works now, we can have lists:

+ first item
+ second item
+ third item

[Links works too](http://localhost)
`

		text4 = `This has whitespace sensitive markdown in the description

|+ first item
|    + nested item
|    + also nested item

Sample code block:

|    fmt.Println("Hello World!")

`
	)

	var err error

	st := &SectionedParser{}
	st.setTitle = func(_ []string) {}
	err = st.Parse(ascg(text))
	require.NoError(t, err)

	assert.Equal(t, []string{"This has a title, separated by a whitespace line"}, st.Title())
	assert.Equal(t, []string{"In this example the punctuation for the title should not matter for swagger.", "For go it will still make a difference though."}, st.Description())

	st = &SectionedParser{}
	st.setTitle = func(_ []string) {}
	err = st.Parse(ascg(text2))
	require.NoError(t, err)

	assert.Equal(t, []string{"This has a title without whitespace."}, st.Title())
	assert.Equal(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())

	st = &SectionedParser{}
	st.setTitle = func(_ []string) {}
	err = st.Parse(ascg(text3))
	require.NoError(t, err)

	assert.Equal(t, []string{"This has a title, and markdown in the description"}, st.Title())
	assert.Equal(t, []string{
		"See how markdown works now, we can have lists:", "",
		"+ first item", "+ second item", "+ third item", "",
		"[Links works too](http://localhost)",
	}, st.Description())

	st = &SectionedParser{}
	st.setTitle = func(_ []string) {}
	err = st.Parse(ascg(text4))
	require.NoError(t, err)

	assert.Equal(t, []string{"This has whitespace sensitive markdown in the description"}, st.Title())
	assert.Equal(t, []string{"+ first item", "    + nested item", "    + also nested item", "", "Sample code block:", "", "    fmt.Println(\"Hello World!\")"}, st.Description())
}

type schemaValidations struct {
	current *spec.Schema
}

func (sv schemaValidations) SetMaximum(val float64, exclusive bool) {
	sv.current.Maximum = &val
	sv.current.ExclusiveMaximum = exclusive
}

func (sv schemaValidations) SetMinimum(val float64, exclusive bool) {
	sv.current.Minimum = &val
	sv.current.ExclusiveMinimum = exclusive
}
func (sv schemaValidations) SetMultipleOf(val float64) { sv.current.MultipleOf = &val }
func (sv schemaValidations) SetMinItems(val int64)     { sv.current.MinItems = &val }
func (sv schemaValidations) SetMaxItems(val int64)     { sv.current.MaxItems = &val }
func (sv schemaValidations) SetMinLength(val int64)    { sv.current.MinLength = &val }
func (sv schemaValidations) SetMaxLength(val int64)    { sv.current.MaxLength = &val }
func (sv schemaValidations) SetPattern(val string)     { sv.current.Pattern = val }
func (sv schemaValidations) SetUnique(val bool)        { sv.current.UniqueItems = val }
func (sv schemaValidations) SetDefault(val any)        { sv.current.Default = val }
func (sv schemaValidations) SetExample(val any)        { sv.current.Example = val }
func (sv schemaValidations) SetEnum(val string) {
	var typ string
	if len(sv.current.Type) > 0 {
		typ = sv.current.Type[0]
	}
	sv.current.Enum = ParseEnum(val, &spec.SimpleSchema{Format: sv.current.Format, Type: typ})
}

func dummybuilder() schemaValidations {
	return schemaValidations{new(spec.Schema)}
}

func TestSectionedParser_TagsDescription(t *testing.T) {
	const (
		block = `This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.
minimum: 10
maximum: 20
`
		block2 = `This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.

minimum: 10
maximum: 20
`
	)

	var err error

	st := &SectionedParser{}
	st.setTitle = func(_ []string) {}
	st.taggers = []TagParser{
		{"Maximum", false, false, nil, &SetMaximum{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, false, nil, &SetMinimum{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, false, nil, &SetMultipleOf{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	err = st.Parse(ascg(block))
	require.NoError(t, err)
	assert.Equal(t, []string{"This has a title without whitespace."}, st.Title())
	assert.Equal(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, 2)
	_, ok := st.matched["Maximum"]
	assert.TrueT(t, ok)
	_, ok = st.matched["Minimum"]
	assert.TrueT(t, ok)

	st = &SectionedParser{}
	st.setTitle = func(_ []string) {}
	st.taggers = []TagParser{
		{"Maximum", false, false, nil, &SetMaximum{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, false, nil, &SetMinimum{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, false, nil, &SetMultipleOf{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	err = st.Parse(ascg(block2))
	require.NoError(t, err)
	assert.Equal(t, []string{"This has a title without whitespace."}, st.Title())
	assert.Equal(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, 2)
	_, ok = st.matched["Maximum"]
	assert.TrueT(t, ok)
	_, ok = st.matched["Minimum"]
	assert.TrueT(t, ok)
}

func TestSectionedParser_Empty(t *testing.T) {
	const block = `swagger:response someResponse`

	var err error

	st := &SectionedParser{}
	st.setTitle = func(_ []string) {}
	ap := newSchemaAnnotationParser("SomeResponse")
	ap.rx = rxResponseOverride
	st.annotation = ap

	err = st.Parse(ascg(block))
	require.NoError(t, err)
	assert.Empty(t, st.Title())
	assert.Empty(t, st.Description())
	assert.Empty(t, st.taggers)
	assert.EqualT(t, "SomeResponse", ap.GoName)
	assert.EqualT(t, "someResponse", ap.Name)
}

func testSectionedParserWithBlock(
	t *testing.T,
	block string,
	expectedMatchedCount int,
	maximumExpected bool,
) {
	t.Helper()

	st := &SectionedParser{}
	st.setTitle = func(_ []string) {}
	ap := newSchemaAnnotationParser("SomeModel")
	st.annotation = ap
	st.taggers = []TagParser{
		{"Maximum", false, false, nil, &SetMaximum{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMaximumFmt, ""))}},
		{"Minimum", false, false, nil, &SetMinimum{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMinimumFmt, ""))}},
		{"MultipleOf", false, false, nil, &SetMultipleOf{builder: dummybuilder(), rx: regexp.MustCompile(fmt.Sprintf(rxMultipleOfFmt, ""))}},
	}

	err := st.Parse(ascg(block))
	require.NoError(t, err)
	assert.Equal(t, []string{"This has a title without whitespace."}, st.Title())
	assert.Equal(t, []string{"The punctuation here does indeed matter. But it won't for go."}, st.Description())
	assert.Len(t, st.matched, expectedMatchedCount)
	_, ok := st.matched["Maximum"]
	assert.EqualT(t, maximumExpected, ok)
	_, ok = st.matched["Minimum"]
	assert.TrueT(t, ok)
	assert.EqualT(t, "SomeModel", ap.GoName)
	assert.EqualT(t, "someModel", ap.Name)
}

func TestSectionedParser_SkipSectionAnnotation(t *testing.T) {
	const block = `swagger:model someModel

This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.

minimum: 10
maximum: 20
`
	testSectionedParserWithBlock(t, block, 2, true)
}

func TestSectionedParser_TerminateOnNewAnnotation(t *testing.T) {
	const block = `swagger:model someModel

This has a title without whitespace.
The punctuation here does indeed matter. But it won't for go.

minimum: 10
swagger:meta
maximum: 20
`
	testSectionedParserWithBlock(t, block, 1, false)
}

func TestSectionedParser_NilDoc(t *testing.T) {
	st := NewSectionedParser(
		WithSetTitle(func(_ []string) {}),
		WithSetDescription(func(_ []string) {}),
	)
	require.NoError(t, st.Parse(nil))
	assert.Empty(t, st.Title())
	assert.Empty(t, st.Description())
	assert.FalseT(t, st.Ignored())
}

func TestSectionedParser_IgnoredAnnotation(t *testing.T) {
	const block = `swagger:ignore SomeType

This should not matter.
`
	st := NewSectionedParser(
		WithSetTitle(func(_ []string) {}),
	)
	err := st.Parse(ascg(block))
	require.NoError(t, err)
	assert.TrueT(t, st.Ignored())
}

func TestSectionedParser_WithoutSetTitle(t *testing.T) {
	// When setTitle is nil, collectTitleDescription cleans up headers
	// but does not split title from description.
	const block = `Just a description line.
Another line.
`
	st := &SectionedParser{}
	err := st.Parse(ascg(block))
	require.NoError(t, err)
	assert.Nil(t, st.Title())
	assert.Equal(t, []string{"Just a description line.", "Another line."}, st.Description())
}

func TestSectionedParser_TagParseError(t *testing.T) {
	// When a matched tagger's Parse returns an error, SectionedParser.Parse propagates it.
	errParser := &failingParser{}
	st := NewSectionedParser(
		WithSetTitle(func(_ []string) {}),
		WithTaggers(
			NewSingleLineTagParser("Failing", errParser),
		),
	)

	const block = `Title.

minimum: 10
`
	err := st.Parse(ascg(block))
	require.Error(t, err)
	assert.ErrorIs(t, err, errForced)
}

type failingParser struct{}

var errForced = errors.New("forced error")

func (f *failingParser) Matches(line string) bool { return rxMinimum.MatchString(line) }
func (f *failingParser) Parse(_ []string) error   { return errForced }

func TestSectionedParser_AnnotationMatchWithHeader(t *testing.T) {
	// When the annotation matches and headers have been collected,
	// seenTag is set to true — further non-tag lines are skipped.
	const block = `swagger:model someModel

Title.
Description.

swagger:model anotherModel
This line after a re-match should still be part of the description.
`
	ap := newSchemaAnnotationParser("SomeModel")
	st := &SectionedParser{}
	st.setTitle = func(_ []string) {}
	st.annotation = ap

	err := st.Parse(ascg(block))
	require.NoError(t, err)
	assert.EqualT(t, "someModel", ap.Name)
}

func ascg(txt string) *ast.CommentGroup {
	var cg ast.CommentGroup
	for line := range strings.SplitSeq(txt, "\n") {
		var cmt ast.Comment
		cmt.Text = "// " + line
		cg.List = append(cg.List, &cmt)
	}
	return &cg
}
