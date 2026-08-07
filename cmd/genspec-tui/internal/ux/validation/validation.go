// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package validation runs the produced spec through go-openapi/validate and normalises what comes back into findings
// the TUI can list and navigate to.
//
// This asks a different question from the scanner's own diagnostics. Those say whether the ANNOTATIONS were understood;
// these say whether the DOCUMENT they produced is a legal Swagger 2.0 spec. A scan can be perfectly clean and still
// emit something a consumer will reject, which is the gap this closes.
package validation

import (
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// Finding is one validation result.
//
// Severity reuses the scanner's own enum rather than declaring a parallel one: the two kinds of finding are shown by
// the same pane in the same colours, and giving "error" two incompatible spellings would only mean translating between
// them at the one place they meet.
type Finding struct {
	Severity grammar.Severity

	// Path is the location the validator reported, in its own dotted notation (`paths./pets.get.parameters.type`).
	// Shown verbatim, because it is what the validator would print and what a reader will search the spec for.
	Path string

	// Pointer is Path converted to RFC 6901, for navigation. Empty when the finding named no location at all.
	Pointer string

	Message string
}

// Run validates a rendered JSON spec.
//
// Takes the rendered bytes rather than the *spec.Swagger the scan produced, so what is checked is exactly the document
// on screen — including whatever the JSON round-trip did to it.
func Run(specJSON []byte) ([]Finding, error) {
	if len(specJSON) == 0 {
		return nil, nil
	}

	doc, err := loads.Analyzed(specJSON, "")
	if err != nil {
		return nil, fmt.Errorf("cannot load the generated spec: %w", err)
	}

	errs, warns := validate.NewSpecValidator(doc.Schema(), strfmt.Default).Validate(doc)

	findings := make([]Finding, 0, len(errs.Errors)+len(warns.Warnings))
	for _, e := range errs.Errors {
		findings = append(findings, finding(grammar.SeverityError, e))
	}
	for _, w := range warns.Warnings {
		findings = append(findings, finding(grammar.SeverityWarning, w))
	}

	return findings, nil
}

// finding normalises one validator error.
func finding(severity grammar.Severity, err error) Finding {
	path := locationOf(err)

	return Finding{
		Severity: severity,
		Path:     path,
		Pointer:  pointerFor(path),
		Message:  err.Error(),
	}
}

// locationOf extracts the location a validator error names.
//
// A *errors.Validation carries it as a field, which is exact. Everything else only has it inside the message, where the
// validator writes it in double quotes at the front ("paths./pets.get.responses.200" must validate…) — so that is read
// back rather than guessed at, and anything not in that shape simply has no location.
func locationOf(err error) string {
	var v *errors.Validation
	if stderrors.As(err, &v) {
		return v.Name
	}

	msg := err.Error()
	if !strings.HasPrefix(msg, `"`) {
		return ""
	}
	if end := strings.Index(msg[1:], `"`); end > 0 {
		return msg[1 : end+1]
	}

	return ""
}

// pointerFor converts the validator's dotted path into an RFC 6901 JSON pointer.
//
// Exact for an ordinary object path. Two things it cannot recover, both properties of the notation rather than of this
// conversion, and both measured rather than assumed (see TestValidation_PointerResolutionAccuracy):
//
//   - the validator omits ARRAY INDICES, reporting `paths./pets.get.parameters.type` for a node that lives at
//     `…/parameters/0/type`. Since parameter lists are always arrays, this is the common case, not an edge one;
//   - a `required` finding names the very node that is missing, so there is nothing to point at.
//
// A path template containing a dot would split wrongly too, though that has not been seen in practice.
//
// All three are why the caller resolves by walking UP to the nearest ancestor that exists: an imprecise result is
// always an ancestor of what was reported, never a sibling.
func pointerFor(path string) string {
	if path == "" {
		return ""
	}

	var b strings.Builder
	for seg := range strings.SplitSeq(path, ".") {
		if seg == "" {
			continue
		}
		b.WriteByte('/')
		// RFC 6901 escaping, and it matters here: every path template contains a slash.
		b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(seg))
	}

	return b.String()
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
