// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// What -validate reports goes through the same reporter as the scan's own findings, so these check both halves: what
// comes back to the command, and what the reader is told.
func TestValidateSpec(t *testing.T) {
	t.Parallel()

	t.Run("should pass a document that is valid", func(t *testing.T) {
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})

		require.NoError(t, report.ValidateSpec([]byte(validSpec)))
		assert.Empty(t, out.String(), "a valid document is not news")
	})

	t.Run("should report what is wrong, and how much", func(t *testing.T) {
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})

		err := report.ValidateSpec([]byte(`{"swagger":"2.0"}`))

		require.ErrorIs(t, err, sentinel.ErrInvalidSpec)
		assert.Contains(t, err.Error(), "finding(s)", "the count, so a caller knows whether it is one thing or ten")
		assert.Contains(t, out.String(), "info in body is required")
		assert.Contains(t, out.String(), "ERRO")
	})

	t.Run("should locate a finding about the document itself", func(t *testing.T) {
		// An empty pointer is a location, not the absence of one - and printed as nothing at all it would read
		// like a finding nobody can place.
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})

		_ = report.ValidateSpec([]byte(`{"swagger":"2.0"}`))

		assert.Contains(t, out.String(), rootPointer)
	})

	t.Run("should count towards the threshold like any other finding", func(t *testing.T) {
		// Policy is about the document, not about which half of the command noticed: a reader meets them as one
		// stream, and a threshold that saw only the scan's own findings would be a trap.
		t.Parallel()

		report, _ := reporterFor(t, ReporterConfig{FailThreshold: codescan.SeverityError, Failing: true})

		_ = report.ValidateSpec([]byte(`{"swagger":"2.0"}`))

		require.ErrorIs(t, report.Summarize(), sentinel.ErrDiagnostics)
	})

	t.Run("should report a warning without failing on it", func(t *testing.T) {
		// A document can be valid and still be worth saying something about - here, a definition nothing refers to.
		// Whether that fails the command is -fail-on's business, not the validator's.
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true, FailThreshold: codescan.SeverityError})

		require.NoError(t, report.ValidateSpec([]byte(specWithAnUnusedDefinition)))

		assert.Contains(t, out.String(), "WARN")
		require.NoError(t, report.Summarize(), "a warning is below the threshold this one was given")
	})

	t.Run("should refuse what it cannot re-read", func(t *testing.T) {
		// Reached only by a document that never came from the renderer, which is worth saying plainly rather than
		// reporting as a specification with a great deal wrong with it.
		t.Parallel()

		report, _ := reporterFor(t, ReporterConfig{Failing: true})

		err := report.ValidateSpec([]byte("not a document at all"))

		require.Error(t, err)
		require.NotErrorIs(t, err, sentinel.ErrInvalidSpec, "the document was never read, so nothing was found in it")
		assert.Contains(t, err.Error(), "cannot re-read the document")
	})
}

// specWithAnUnusedDefinition is valid, and has something to say about itself: nothing refers to Pet.
const specWithAnUnusedDefinition = `{
  "swagger": "2.0",
  "info": {"title": "Pets", "version": "1.0"},
  "paths": {
    "/pets": {
      "get": {
        "responses": {"200": {"description": "every pet"}}
      }
    }
  },
  "definitions": {
    "Pet": {"type": "object", "properties": {"name": {"type": "string"}}}
  }
}`

// validSpec is the smallest Swagger 2.0 document that validates: an info block, and a path so that the document
// describes something.
const validSpec = `{
  "swagger": "2.0",
  "info": {"title": "Pets", "version": "1.0"},
  "paths": {
    "/pets": {
      "get": {
        "responses": {"200": {"description": "every pet"}}
      }
    }
  }
}`
