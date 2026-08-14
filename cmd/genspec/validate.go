// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log/slog"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"

	"github.com/go-openapi/codescan"
)

// rootPointer names what the empty JSON pointer addresses.
//
// An empty pointer is a location, not the absence of one: RFC 6901 spells the whole document that
// way, and a finding about something the document does not have at all - no info block, no paths -
// is reported there. Printed as prose because "" would render as nothing at all.
const rootPointer = "(the whole document)"

// checkValid reports what is wrong with the document the scan produced.
//
// It runs on the rendered JSON rather than on the spec.Swagger it came from, so what is checked is
// exactly what was written - including whatever the round-trip through JSON did to it.
//
// Findings go through the same reporter as the scan's own diagnostics: to a reader they are the same
// kind of news about the same document, and giving them a second dialect would only mean two things
// to learn.
func checkValid(asJSON []byte, report *reporter) error {
	document, err := loads.Analyzed(asJSON, "")
	if err != nil {
		return fmt.Errorf("cannot re-read the document to validate it: %w", err)
	}

	// Everything is read off the FIRST result, which holds both lists. The second is a warnings-only
	// view of the same run, and it carries them as its errors - so reading warnings from its Warnings
	// field finds an empty slice, and a document with warnings reports as clean.
	result, _ := validate.NewSpecValidator(document.Schema(), strfmt.Default).Validate(document)

	errored := result.LocatedErrors()
	for _, located := range errored {
		report.finding(codescan.SeverityError, located)
	}
	for _, located := range result.LocatedWarnings() {
		report.finding(codescan.SeverityWarning, located)
	}

	if len(errored) > 0 {
		return fmt.Errorf("%w: %d finding(s)", errInvalidSpec, len(errored))
	}

	return nil
}

// finding reports one validation result, located by the pointer the validator recorded.
//
// The pointer is carried straight through rather than recovered from the message: the validator
// records where it was as it walks, so it is the authority on it, and a sentence is a poor place to
// keep a machine-readable path.
func (r *reporter) finding(severity codescan.Severity, located validate.Located) {
	r.record(severity)

	pointer := located.Pointer
	if pointer == "" {
		pointer = rootPointer
	}

	r.log(severity, located.Err.Error(), slog.String("at", pointer))
}
