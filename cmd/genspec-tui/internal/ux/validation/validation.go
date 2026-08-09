// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// RootLabel names what the empty pointer addresses.
//
// An empty pointer is a location, not the absence of one: RFC 6901 spells the whole document that way, and a finding
// about something the document does not have at all - no info block, no paths - is reported there. It needs a printable
// stand-in, since "" would render as nothing at all.
const RootLabel = "(the whole document)"

// Finding is one validation result.
//
// Severity reuses the scanner's own enum rather than declaring a parallel one: the two kinds of finding are shown by
// the same pane in the same colours, and giving "error" two incompatible spellings would only mean translating between
// them at the one place they meet.
type Finding struct {
	Severity grammar.Severity

	// Pointer is where the validator says the offending value is, as an RFC 6901 JSON pointer.
	//
	// Taken from the result rather than recovered from the message: the validator records the location as it walks, so
	// it is the authority on it, and a sentence is a poor place to keep a machine-readable path.
	//
	// EMPTY means the whole document, which is what RFC 6901 spells that way - not "nowhere". A finding about
	// something the document lacks entirely is reported there, so an empty pointer is navigable: see RootLabel.
	Pointer string

	Message string
}

// Run validates a rendered JSON spec.
//
// Takes the rendered bytes rather than the *spec.Swagger the scan produced, so what is checked is exactly the document
// on screen - including whatever the JSON round-trip did to it.
func Run(specJSON []byte) ([]Finding, error) {
	if len(specJSON) == 0 {
		return nil, nil
	}

	doc, err := loads.Analyzed(specJSON, "")
	if err != nil {
		return nil, fmt.Errorf("cannot load the generated spec: %w", err)
	}

	// Everything comes off the FIRST result, which holds both lists. The second is a warnings-only view of the same
	// run, and it carries them as its ERRORS - so reading warnings from its Warnings field found an empty slice and the
	// pane reported a spec with warnings as clean.
	result, _ := validate.NewSpecValidator(doc.Schema(), strfmt.Default).Validate(doc)

	locatedErrors, locatedWarnings := result.LocatedErrors(), result.LocatedWarnings()

	findings := make([]Finding, 0, len(locatedErrors)+len(locatedWarnings))
	for _, l := range locatedErrors {
		findings = append(findings, finding(grammar.SeverityError, l))
	}
	for _, l := range locatedWarnings {
		findings = append(findings, finding(grammar.SeverityWarning, l))
	}

	return findings, nil
}

// finding normalises one located validator result.
//
// The pointer is carried straight through: it is already RFC 6901, and it comes from the validator's own record of
// where it was when the check failed. Deriving it from the message instead - which is what this did - could not recover
// an array index, since the notation there has no place to put one, and a parameter list is always an array.
func finding(severity grammar.Severity, located validate.Located) Finding {
	return Finding{
		Severity: severity,
		Pointer:  located.Pointer,
		Message:  located.Err.Error(),
	}
}

// Tally counts findings by severity, for the pane's summary line.
func Tally(findings []Finding) (errs, warns int) {
	for _, f := range findings {
		if f.Severity == grammar.SeverityError {
			errs++

			continue
		}
		warns++
	}

	return errs, warns
}
