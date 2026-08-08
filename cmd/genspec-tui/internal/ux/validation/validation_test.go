// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// brokenSpec carries one fault per validator behaviour worth covering.
//
// A bad parameter type, a duplicate operationId, and a response missing its required description.
const brokenSpec = `{
  "swagger": "2.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "getPets",
        "parameters": [{"name": "limit", "in": "query", "type": "nope"}],
        "responses": {"200": {}}
      }
    }
  }
}`

const validSpec = `{
  "swagger": "2.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/pets": {
      "get": {"operationId": "getPets", "responses": {"200": {"description": "ok"}}}
    }
  }
}`

// warnSpec is valid but carries things nothing refers to, which the validator reports as WARNINGS rather than errors.
//
// The distinction is the point: a spec can be legal and still worth saying something about.
const warnSpec = `{
  "swagger": "2.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/pets": {
      "get": {"operationId": "getPets", "responses": {"200": {"description": "ok"}}}
    }
  },
  "definitions": {"NeverUsed": {"type": "object"}},
  "parameters": {"unusedParam": {"name": "q", "in": "query", "type": "string"}}
}`

// TestRun_ReportsWarnings covers the half of the validator's output that is not errors.
//
// It is a regression test with a story: warnings were read off the warnings-ONLY result, whose warnings live in its
// Errors field, so the slice was always empty and a spec with warnings was reported as clean.
func TestRun_ReportsWarnings(t *testing.T) {
	findings, err := Run([]byte(warnSpec))
	require.NoError(t, err)

	var warns []Finding
	for _, f := range findings {
		if f.Severity == grammar.SeverityWarning {
			warns = append(warns, f)
		}
	}

	require.NotEmpty(t, warns, "a spec with unreferenced components must not read as clean")
	for _, f := range warns {
		assert.NotEmpty(t, f.Message)
		assert.NotEmpty(t, f.Pointer, "a warning is navigable too: %q", f.Message)
	}
	// These name their subject inside the sentence ("#/definitions/NeverUsed is not used anywhere"), so a location read
	// out of the message would find nothing here. The validator's own record has it.

	errsOnly, _ := Tally(findings)
	assert.Zero(t, errsOnly, "the spec itself is legal; nothing here is an error")
}

func TestRun_ReportsFindings(t *testing.T) {
	findings, err := Run([]byte(brokenSpec))

	require.NoError(t, err)
	require.NotEmpty(t, findings)

	// Every finding must be usable: something to read, and a severity the pane can colour.
	for _, f := range findings {
		assert.NotEmpty(t, f.Message)
		assert.Contains(t, []grammar.Severity{grammar.SeverityError, grammar.SeverityWarning}, f.Severity)
	}
}

// TestRun_LocatesFindings pins that findings carry somewhere to go.
//
// That is what makes the tab navigable rather than merely readable.
func TestRun_LocatesFindings(t *testing.T) {
	findings, err := Run([]byte(brokenSpec))
	require.NoError(t, err)

	located := 0
	for _, f := range findings {
		if f.Pointer != "" {
			located++
			assert.True(t, f.Pointer[0] == '/', "a pointer starts at the root: %q", f.Pointer)
		}
	}

	assert.Positive(t, located, "no finding could be located in the spec; the tab would not navigate at all")
}

func TestRun_ValidSpecHasNoFindings(t *testing.T) {
	findings, err := Run([]byte(validSpec))

	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_EmptyInput(t *testing.T) {
	findings, err := Run(nil)

	require.NoError(t, err)
	assert.Empty(t, findings, "there is nothing to say about a spec that has not been generated")
}

func TestRun_UnloadableSpec(t *testing.T) {
	_, err := Run([]byte("{not json"))

	require.Error(t, err, "a document that cannot even be loaded is reported, not silently valid")
}

func TestTally(t *testing.T) {
	errs, warns := Tally([]Finding{
		{Severity: grammar.SeverityError},
		{Severity: grammar.SeverityWarning},
		{Severity: grammar.SeverityError},
	})

	assert.Equal(t, 2, errs)
	assert.Equal(t, 1, warns)
}
