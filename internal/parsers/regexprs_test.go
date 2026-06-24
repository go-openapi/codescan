// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestRouteExpression(t *testing.T) {
	assert.RegexpT(t, rxRoute, "swagger:route DELETE /orders/{id} deleteOrder")
	assert.RegexpT(t, rxRoute, "swagger:route GET /v1.2/something deleteOrder")
}

func TestOperationExpression(t *testing.T) {
	assert.RegexpT(t, rxOperation, "swagger:operation DELETE /orders/{id} deleteOrder")
	assert.RegexpT(t, rxOperation, "swagger:operation GET /v1.2/something deleteOrder")
}

func TestSchemaValueExtractors(t *testing.T) {
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

	const numValid = 4 + 3 // +3 extra space
	validParams := make([]string, 0, numValid)
	validParams = append(validParams,
		"yada123",
		"date",
		"date-time",
		"long-combo-1-with-combo-2-and-a-3rd-one-too",
	)
	invalidParams := []string{
		"1-yada-3",
		"1-2-3",
		"-yada-3",
		"-2-3",
		"*blah",
		"blah*",
	}

	verifySwaggerOneArgSwaggerTag(t, rxModelOverride, models, append(validParams, "", "  ", " "), invalidParams)
}

// TestParametersClassificationGate verifies that rxParametersOverride is a
// PERMISSIVE presence gate: it matches `swagger:parameters` plus any
// non-empty argument and captures it verbatim, leaving shape validation to
// the grammar. Forms the strict model matcher rejects (a leading `*`, a
// `/path`, malformed idents) are accepted here so the grammar can parse
// and diagnose them. A bare keyword with no argument is not classified.
func TestParametersClassificationGate(t *testing.T) {
	prefixes := []string{
		"// swagger:parameters ",
		"swagger:parameters ",
		"* swagger:parameters    ",
	}
	accepted := []string{
		"listPets",              // operation id
		"listPets createPet",    // multiple operation ids
		"*",                     // shared-namespace target
		"* listPets",            // shared register + op id
		"/pets",                 // path target
		"/pets X-Request-ID",    // path reference
		"listPets X-Request-ID", // operation reference
		"*blah",                 // malformed — accepted; the grammar's concern
		"1-2-3",                 // malformed — accepted; the grammar's concern
	}
	for _, pref := range prefixes {
		for _, arg := range accepted {
			line := pref + arg
			m := rxParametersOverride.FindStringSubmatch(line)
			if !assert.Len(t, m, 2) {
				t.Logf("expected %q to be classified", line)
				continue
			}
			assert.EqualT(t, strings.TrimSpace(arg), m[1])
		}
	}

	// A bare keyword with no argument is not classified (the grammar's
	// missing-target diagnostic fires only once a node is classified).
	for _, line := range []string{"swagger:parameters", "swagger:parameters ", "// swagger:parameters   "} {
		assert.Empty(t, rxParametersOverride.FindStringSubmatch(line), "bare keyword must not classify")
	}
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
