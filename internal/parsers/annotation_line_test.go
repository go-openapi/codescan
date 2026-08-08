// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The regular expressions the string searches in annotation_line.go replaced.
//
// They are kept HERE, and only here, as the specification being reproduced: an equivalence claim
// needs both sides present to be checkable, and a rule stated twice in prose is not a check. Nothing
// outside this file may use them.
const refCommentPrefix = `^[\p{Zs}\t/\*-]*\|?\p{Zs}*`

var (
	refSwaggerAnnotation  = regexp.MustCompile(`(?:^|[\s/])swagger:([\p{L}\p{N}\p{Pd}\p{Pc}]+)`)
	refModelOverride      = regexp.MustCompile(refCommentPrefix + `swagger:model\p{Zs}*(\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)?(?:\.)?$`)
	refResponseOverride   = regexp.MustCompile(refCommentPrefix + `swagger:response\p{Zs}*(\*|\p{L}[\p{L}\p{N}\p{Pd}\p{Pc}]+)?(?:\.)?$`)
	refParametersOverride = regexp.MustCompile(refCommentPrefix + `swagger:parameters\p{Zs}+(\S.*?)\p{Zs}*$`)
	refModelArg           = regexp.MustCompile(refCommentPrefix + `swagger:model\p{Zs}+(\S.*?)\p{Zs}*$`)
	refResponseArg        = regexp.MustCompile(refCommentPrefix + `swagger:response\p{Zs}+(\S.*?)\p{Zs}*$`)
)

// annotationLineCorpus is every line shape the classifiers are asked about: leading comment noise,
// a marker, a separator, an argument, trailing noise.
//
// The pieces are chosen for the boundaries the rules turn on — the single markdown table pipe, a
// tab (which is comment noise but not a space separator), a non-breaking space (a space separator
// but not ASCII whitespace), a one-character name, a package-qualified name, a trailing sentence
// period, and prose in front of the marker.
func annotationLineCorpus() []string {
	prefixes := []string{
		"", "//", "// ", "  // ", "\t", "\t* ", " * ", "- ", "* ",
		"|", "| ", " | ", "|| ", "/* ", "  ", " ", " | ",
		"x ", "DoBad ", "prose about ", "// | ", "//|swagger ", "-|-",
	}
	markers := []string{
		"swagger:model", "swagger:response", "swagger:parameters",
		"swagger:modelfoo", "swagger:models", "swagger:responses", "swagger:route",
		"swagger:", "swagger", "Swagger:model",
	}
	separators := []string{"", " ", "  ", "\t", " \t", " ", "   ", "   "}
	args := []string{
		"", "Foo", "F", "utils.Error", "*", "**", "*blah", "blah*", "1-2-3", "-yada",
		"Foo.", "Foo..", "a-b_c1", "listPets createPet", "/pets", "Ünïcode", "\u65e5\u672c\u8a9e", // a non-Latin script argument
		"x", "swagger:model Bar", "a b", "-", "_", "A_1",
	}
	suffixes := []string{"", " ", "  ", "\t", "\r", " ", " ."}

	out := make([]string, 0, len(prefixes)*len(markers)*len(separators)*len(args)*len(suffixes))
	for _, prefix := range prefixes {
		for _, marker := range markers {
			for _, sep := range separators {
				for _, arg := range args {
					for _, suffix := range suffixes {
						out = append(out, prefix+marker+sep+arg+suffix)
					}
				}
			}
		}
	}

	return out
}

// TestAnnotationLine_ReproducesTheRetiredExpressions is the equivalence proof behind replacing the
// classification regexes with string searches.
//
// Every classifier is run against its retired expression over the whole line corpus, capture
// included — the rules are subtle enough (a name of one character is not a name; a marker may carry
// a trailing period; a tab is comment noise but not a separator) that agreeing on the verdict but
// not on the captured name would be a real regression.
//
// The argument matchers have one documented divergence, covered by its own test below and excluded
// here by argumentComparable: see annotationArgument.
func TestAnnotationLine_ReproducesTheRetiredExpressions(t *testing.T) {
	t.Parallel()

	corpus := annotationLineCorpus()
	require.Greater(t, len(corpus), 10000)

	for _, line := range corpus {
		gotName, gotOK := annotationName(line)
		wantName, wantOK := referenceSubmatch(refSwaggerAnnotation, line)
		assert.EqualT(t, wantOK, gotOK, "annotationName verdict on %q", line)
		assert.EqualT(t, wantName, gotName, "annotationName capture on %q", line)

		assertOverride(t, line, keywordModel, false, refModelOverride)
		assertOverride(t, line, keywordResponse, true, refResponseOverride)

		assertArgument(t, line, keywordModel, refModelArg)
		assertArgument(t, line, keywordResponse, refResponseArg)
		assertArgument(t, line, keywordParameters, refParametersOverride)
	}
}

// TestAnnotationLine_ArgumentStartingOnASeparator locks the one place the string search reads
// differently from the expression it replaced.
//
// `\S` excludes the ASCII whitespace characters and nothing else, so a non-breaking space passes
// it. When what followed the separator run was a tab (or nothing), the old pattern gave a
// separator back and started the argument on it. The argument now begins where the separators end,
// or there is no argument.
func TestAnnotationLine_ArgumentStartingOnASeparator(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		line string
		was  string
	}{
		{line: "// swagger:parameters \u00a0", was: "\u00a0"},
		{line: "// swagger:parameters \u00a0\t", was: "\u00a0\t"},
		{line: "swagger:response \u00a0\t0", was: "\u00a0\t0"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			reference := refParametersOverride
			keyword := keywordParameters
			if strings.Contains(tc.line, "swagger:response") {
				reference, keyword = refResponseArg, keywordResponse
			}

			was, wasOK := referenceSubmatch(reference, tc.line)
			require.TrueT(t, wasOK)
			assert.EqualT(t, tc.was, was, "the retired expression started the argument on a separator")

			_, ok := annotationArgument(tc.line, keyword)
			assert.FalseT(t, ok, "an argument does not start on a space separator")

			// The marker is still classified as a marker: only its argument reads differently.
			_, ok = annotationName(tc.line)
			assert.TrueT(t, ok)
		})
	}
}

// argumentComparable reports whether line falls outside the documented divergence.
//
// The two readings can only part company over WHERE the argument starts, and the string search
// never starts one on a space separator. So a retired expression whose capture begins with a
// separator is the divergence, and every other line must agree exactly.
func argumentComparable(line string, reference *regexp.Regexp) bool {
	captured, ok := referenceSubmatch(reference, line)
	if !ok || captured == "" {
		return true
	}
	first, _ := utf8.DecodeRuneInString(captured)

	return !isSpaceSeparator(first)
}

func FuzzAnnotationLine(f *testing.F) {
	// A sample of the corpus rather than all of it: the exhaustive comparison is the test above, and
	// a seed set of a quarter of a million lines leaves the fuzzer no time to mutate any of them.
	corpus := annotationLineCorpus()
	for i := 0; i < len(corpus); i += 211 {
		f.Add(corpus[i])
	}
	for _, line := range []string{
		"// swagger:model Foo", "// swagger:response *", "// swagger:parameters a b",
		"|\tswagger:model Foo.", "// see swagger:model Foo", "swagger:model\u00a0Foo",
	} {
		f.Add(line)
	}

	f.Fuzz(func(t *testing.T, line string) {
		// The classifiers are given one comment line at a time; a newline never reaches them and the
		// retired expressions were never asked about one.
		if strings.ContainsAny(line, "\n") {
			t.Skip()
		}

		gotName, gotOK := annotationName(line)
		wantName, wantOK := referenceSubmatch(refSwaggerAnnotation, line)
		assert.EqualT(t, wantOK, gotOK, "annotationName verdict on %q", line)
		assert.EqualT(t, wantName, gotName, "annotationName capture on %q", line)

		assertOverride(t, line, keywordModel, false, refModelOverride)
		assertOverride(t, line, keywordResponse, true, refResponseOverride)
		assertArgument(t, line, keywordModel, refModelArg)
		assertArgument(t, line, keywordResponse, refResponseArg)
		assertArgument(t, line, keywordParameters, refParametersOverride)
	})
}

func assertOverride(t *testing.T, line, keyword string, wildcard bool, reference *regexp.Regexp) {
	t.Helper()

	gotName, gotOK := overrideName(line, keyword, wildcard)
	wantName, wantOK := referenceSubmatch(reference, line)
	assert.EqualT(t, wantOK, gotOK, "swagger:%s override verdict on %q", keyword, line)
	assert.EqualT(t, wantName, gotName, "swagger:%s override capture on %q", keyword, line)
}

func assertArgument(t *testing.T, line, keyword string, reference *regexp.Regexp) {
	t.Helper()

	if !argumentComparable(line, reference) {
		return // the documented divergence; see TestAnnotationLine_ArgumentOfNothingButSeparators
	}

	gotArg, gotOK := annotationArgument(line, keyword)
	wantArg, wantOK := referenceSubmatch(reference, line)
	assert.EqualT(t, wantOK, gotOK, "swagger:%s argument verdict on %q", keyword, line)
	assert.EqualT(t, wantArg, gotArg, "swagger:%s argument capture on %q", keyword, line)
}

// referenceSubmatch returns a retired expression's first capture and whether it matched at all.
func referenceSubmatch(rx *regexp.Regexp, line string) (string, bool) {
	m := rx.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}

	return m[1], true
}

// TestAnnotationLine_LineStartRule locks the rule the whole classification hangs on, in the terms
// an author would state it.
func TestAnnotationLine_LineStartRule(t *testing.T) {
	t.Parallel()

	for _, line := range []string{
		"// swagger:model Foo",
		"//swagger:model Foo",
		"  *  swagger:model Foo",
		"\t- swagger:model Foo",
		"| swagger:model Foo", // a markdown table cell
		"swagger:model Foo",
	} {
		name, ok := overrideName(line, keywordModel, false)
		assert.TrueT(t, ok, "%q starts with the marker after comment noise", line)
		assert.EqualT(t, "Foo", name)
	}

	for _, line := range []string{
		"// the swagger:model Foo annotation",
		"// see swagger:model Foo",
		"// || swagger:model Foo", // two pipes: only one is allowed
		"// x swagger:model Foo",
	} {
		_, ok := overrideName(line, keywordModel, false)
		assert.FalseT(t, ok, "%q mentions the marker in prose", line)
	}
}
