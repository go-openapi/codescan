// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestSchemaValueExtractors(t *testing.T) {
	strfmts := []string{
		"// swagger:strfmt ",
		"* swagger:strfmt ",
		"* swagger:strfmt ",
		" swagger:strfmt ",
		"swagger:strfmt ",
		"// swagger:strfmt    ",
		"* swagger:strfmt     ",
		"* swagger:strfmt    ",
		" swagger:strfmt     ",
		"swagger:strfmt      ",
	}
	models := []string{
		"// swagger:model ",
		"* swagger:model ",
		"* swagger:model ",
		" swagger:model ",
		"swagger:model ",
		"// swagger:model    ",
		"* swagger:model     ",
		"* swagger:model    ",
		" swagger:model     ",
		"swagger:model      ",
	}

	allOf := []string{
		"// swagger:allOf ",
		"* swagger:allOf ",
		"* swagger:allOf ",
		" swagger:allOf ",
		"swagger:allOf ",
		"// swagger:allOf    ",
		"* swagger:allOf     ",
		"* swagger:allOf    ",
		" swagger:allOf     ",
		"swagger:allOf      ",
	}

	parameters := []string{
		"// swagger:parameters ",
		"* swagger:parameters ",
		"* swagger:parameters ",
		" swagger:parameters ",
		"swagger:parameters ",
		"// swagger:parameters    ",
		"* swagger:parameters     ",
		"* swagger:parameters    ",
		" swagger:parameters     ",
		"swagger:parameters      ",
	}

	validParams := []string{
		"yada123",
		"date",
		"date-time",
		"long-combo-1-with-combo-2-and-a-3rd-one-too",
	}
	invalidParams := make([]string, 0, 9)
	invalidParams = append(invalidParams,
		"1-yada-3",
		"1-2-3",
		"-yada-3",
		"-2-3",
		"*blah",
		"blah*",
	)

	verifySwaggerOneArgSwaggerTag(t, rxStrFmt, strfmts, validParams, append(invalidParams, "", "  ", " "))
	verifySwaggerOneArgSwaggerTag(t, rxModelOverride, models, append(validParams, "", "  ", " "), invalidParams)

	verifySwaggerOneArgSwaggerTag(t, rxAllOf, allOf, append(validParams, "", "  ", " "), invalidParams)

	verifySwaggerMultiArgSwaggerTag(t, rxParametersOverride, parameters, validParams, invalidParams)

	verifyMinMax(t, rxf(rxMinimumFmt, ""), "min", []string{"", ">", "="})
	verifyMinMax(t, rxf(rxMinimumFmt, fmt.Sprintf(rxItemsPrefixFmt, 1)), "items.min", []string{"", ">", "="})
	verifyMinMax(t, rxf(rxMaximumFmt, ""), "max", []string{"", "<", "="})
	verifyMinMax(t, rxf(rxMaximumFmt, fmt.Sprintf(rxItemsPrefixFmt, 1)), "items.max", []string{"", "<", "="})
	verifyNumeric2Words(t, rxf(rxMultipleOfFmt, ""), "multiple", "of")
	verifyNumeric2Words(t, rxf(rxMultipleOfFmt, fmt.Sprintf(rxItemsPrefixFmt, 1)), "items.multiple", "of")

	verifyIntegerMinMaxManyWords(t, rxf(rxMinLengthFmt, ""), "min", []string{"len", "length"})
	// pattern
	patPrefixes := cartesianJoin(
		[]string{"//", "*", ""},
		[]string{"", " ", "  ", "     "},
		[]string{"pattern", "Pattern"},
		[]string{"", " ", "  ", "     "},
		[]string{":"},
		[]string{"", " ", "  ", "     "},
	)
	verifyRegexpArgs(t, rxf(rxPatternFmt, ""), patPrefixes, []string{"^\\w+$", "[A-Za-z0-9-.]*"}, nil, 2, 1)

	verifyIntegerMinMaxManyWords(t, rxf(rxMinItemsFmt, ""), "min", []string{"items"})
	verifyBoolean(t, rxf(rxUniqueFmt, ""), []string{"unique"}, nil)

	verifyBoolean(t, rxReadOnly, []string{"read"}, []string{"only"})
	verifyBoolean(t, rxRequired, []string{"required"}, nil)
}

func makeMinMax(lower string) (res []string) {
	for _, a := range []string{"", "imum"} {
		res = append(res, lower+a, strings.Title(lower)+a) //nolint:staticcheck // Title is deprecated, yet still useful here. The replacement is bit heavy for just this test
	}

	return res
}

// cartesianJoin returns all concatenations formed by picking one element from each slot.
func cartesianJoin(slots ...[]string) []string {
	result := []string{""}
	for _, slot := range slots {
		next := make([]string, 0, len(result)*len(slot))
		for _, prefix := range result {
			for _, s := range slot {
				next = append(next, prefix+s)
			}
		}
		result = next
	}

	return result
}

// titleCaseVariants returns each name paired with its Title-cased form.
func titleCaseVariants(names []string) []string {
	result := make([]string, 0, len(names)*2)
	for _, nm := range names {
		result = append(result, nm, strings.Title(nm)) //nolint:staticcheck // Title is deprecated, yet still useful here
	}

	return result
}

// verifyRegexpArgs tests that matcher matches lines formed by each prefix+validArg
// (expecting expectedMatchLen matches with the value at matchIdx) and rejects prefix+invalidArg.
func verifyRegexpArgs(t *testing.T, matcher *regexp.Regexp, prefixes, validArgs, invalidArgs []string, expectedMatchLen, matchIdx int) int {
	t.Helper()
	cnt := 0
	for _, prefix := range prefixes {
		for _, vv := range validArgs {
			matches := matcher.FindStringSubmatch(prefix + vv)
			assert.Len(t, matches, expectedMatchLen)
			assert.EqualT(t, vv, matches[matchIdx])
			cnt++
		}

		for _, iv := range invalidArgs {
			matches := matcher.FindStringSubmatch(prefix + iv)
			assert.Empty(t, matches)
			cnt++
		}
	}

	return cnt
}

func verifyBoolean(t *testing.T, matcher *regexp.Regexp, names, names2 []string) {
	t.Helper()

	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validArgs := []string{"true", "false"}
	invalidArgs := []string{"TRUE", "FALSE", "t", "f", "1", "0", "True", "False", "true*", "false*"}

	nms := titleCaseVariants(names)

	var rnms []string
	if len(names2) > 0 {
		nms2 := titleCaseVariants(names2)
		spacesAndDash := []string{"", " ", "  ", "     ", "-"}
		for _, nm := range nms {
			for _, sep := range spacesAndDash {
				for _, nm2 := range nms2 {
					rnms = append(rnms, nm+sep+nm2)
				}
			}
		}
	} else {
		rnms = nms
	}

	linePrefixes := cartesianJoin(prefixes, extraSpaces, rnms, extraSpaces, []string{":"}, extraSpaces)
	cnt := verifyRegexpArgs(t, matcher, linePrefixes, validArgs, invalidArgs, 2, 1)

	var nm2 string
	if len(names2) > 0 {
		nm2 = " " + names2[0]
	}
	t.Logf("tested %d %s%s combinations\n", cnt, names[0], nm2)
}

func verifyIntegerMinMaxManyWords(t *testing.T, matcher *regexp.Regexp, name1 string, words []string) {
	t.Helper()

	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validArgs := []string{"0", "1234"}
	invalidArgs := []string{"1A3F", "2e10", "*12", "12*", "-1235", "0.0", "1234.0394", "-2948.484"}

	wordVariants := titleCaseVariants(words)
	spacesAndDash := []string{"", " ", "  ", "     ", "-"}
	linePrefixes := cartesianJoin(prefixes, extraSpaces, makeMinMax(name1), spacesAndDash, wordVariants, extraSpaces, []string{":"}, extraSpaces)
	cnt := verifyRegexpArgs(t, matcher, linePrefixes, validArgs, invalidArgs, 2, 1)

	var nm2 string
	if len(words) > 0 {
		nm2 = " " + words[0]
	}
	t.Logf("tested %d %s%s combinations\n", cnt, name1, nm2)
}

func verifyNumeric2Words(t *testing.T, matcher *regexp.Regexp, name1, name2 string) {
	t.Helper()

	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validArgs := []string{"0", "1234", "-1235", "0.0", "1234.0394", "-2948.484"}
	invalidArgs := []string{"1A3F", "2e10", "*12", "12*"}

	titleName1 := strings.Title(name1) //nolint:staticcheck // Title is deprecated, yet still useful here
	titleName2 := strings.Title(name2) //nolint:staticcheck // Title is deprecated, yet still useful here
	nameVariants := make([]string, 0, 4*len(extraSpaces))
	for _, es := range extraSpaces {
		nameVariants = append(nameVariants,
			name1+es+name2,
			titleName1+es+titleName2,
			titleName1+es+name2,
			name1+es+titleName2,
		)
	}

	linePrefixes := cartesianJoin(prefixes, extraSpaces, nameVariants, extraSpaces, []string{":"}, extraSpaces)
	cnt := verifyRegexpArgs(t, matcher, linePrefixes, validArgs, invalidArgs, 2, 1)
	t.Logf("tested %d %s %s combinations\n", cnt, name1, name2)
}

func verifyMinMax(t *testing.T, matcher *regexp.Regexp, name string, operators []string) {
	t.Helper()

	extraSpaces := []string{"", " ", "  ", "     "}
	prefixes := []string{"//", "*", ""}
	validArgs := []string{"0", "1234", "-1235", "0.0", "1234.0394", "-2948.484"}
	invalidArgs := []string{"1A3F", "2e10", "*12", "12*"}

	linePrefixes := cartesianJoin(prefixes, extraSpaces, makeMinMax(name), extraSpaces, []string{":"}, extraSpaces, operators, extraSpaces)
	cnt := verifyRegexpArgs(t, matcher, linePrefixes, validArgs, invalidArgs, 3, 2)
	t.Logf("tested %d %s combinations\n", cnt, name)
}

func verifySwaggerOneArgSwaggerTag(t *testing.T, matcher *regexp.Regexp, prefixes, validParams, invalidParams []string) {
	t.Helper()

	for _, pref := range prefixes {
		for _, param := range validParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			if assert.Len(t, matches, 2) {
				assert.EqualT(t, strings.TrimSpace(param), matches[1])
			}
		}
	}

	for _, pref := range prefixes {
		for _, param := range invalidParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			assert.Empty(t, matches)
		}
	}
}

func verifySwaggerMultiArgSwaggerTag(t *testing.T, matcher *regexp.Regexp, prefixes, validParams, invalidParams []string) {
	t.Helper()

	actualParams := make([]string, 0, len(validParams))
	vp := make([]string, 0, len(validParams)+1)

	for i := range validParams {
		vp = vp[:0]
		for j := range i + 1 {
			vp = append(vp, validParams[j]) //nolint:gosec // G602: j is bounded by i+1 which is bounded by len(validParams)
		}

		actualParams = append(actualParams, strings.Join(vp, " "))
	}

	for _, pref := range prefixes {
		for _, param := range actualParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			assert.Len(t, matches, 2)
			assert.EqualT(t, strings.TrimSpace(param), matches[1])
		}
	}

	for _, pref := range prefixes {
		for _, param := range invalidParams {
			line := pref + param
			matches := matcher.FindStringSubmatch(line)
			assert.Empty(t, matches)
		}
	}
}
