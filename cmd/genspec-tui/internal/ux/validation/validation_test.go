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
			assert.NotContains(t, f.Pointer, ".", "the dotted path must have been converted: %q", f.Pointer)
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

// TestPointerFor covers the conversion from the validator's dotted notation to RFC 6901.
//
// Including the escaping that every path template needs.
func TestPointerFor(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"", ""},
		{"paths", "/paths"},
		{"paths./pets.get.responses.200", "/paths/~1pets/get/responses/200"},
		{"definitions.User.properties.email", "/definitions/User/properties/email"},
		// `~` and `/` are the two characters RFC 6901 escapes, and a spec can contain both.
		{"definitions.a~b", "/definitions/a~0b"},
		{"paths./a/b", "/paths/~1a~1b"},
	} {
		assert.Equal(t, tc.want, pointerFor(tc.in), "pointerFor(%q)", tc.in)
	}
}

// TestPointerFor_AmbiguousPathTemplate documents one known limit of the notation.
//
// It cannot express a path template containing a dot, so the conversion splits it.
//
// Recorded for completeness, but it is NOT the limitation that bites in practice - see
// TestValidation_PointerResolutionAccuracy for the two that do, both about what the validator omits rather than about
// what this splits.
func TestPointerFor_AmbiguousPathTemplate(t *testing.T) {
	got := pointerFor("paths./pets.json.get")

	assert.Equal(t, "/paths/~1pets/json/get", got,
		"the dot inside the template is indistinguishable from a separator")
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
