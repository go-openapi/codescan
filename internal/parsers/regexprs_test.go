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
	verifySwaggerMultiArgSwaggerTag(t, rxParametersOverride, parameters, validParams, invalidParams)
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
			vp = append(vp, validParams[j]) // G602 (false positive from gosec now fixed): j is bounded by i+1 which is bounded by len(validParams)
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
