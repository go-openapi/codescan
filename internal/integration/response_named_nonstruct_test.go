// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A `swagger:response` declared on a NAMED type whose underlying is not a struct must render its
// body as the same type does when reached as a model field. Both are full-schema positions, and a
// declaration should not mean something different depending on which one reads it.
//
// Three of these used to emit a response carrying a description and NO SCHEMA AT ALL: the arm
// short-circuited on the stdlib time recognizer and on the declaration's format, and both branches
// wrote into a local schema and returned without the call that attaches it.
//
// The sub-build is deliberately handed the type's UNDERLYING rather than its declaration. A
// `swagger:response` declares a response, not a model, and passing the named type sends it through
// the $ref machinery and publishes it as a definition — which is what the response-toplevel-example
// and response-edges witnesses exist to prevent.
func TestResponseNamedNonStruct(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-named-nonstruct/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Cells where the two positions are legitimately expected to differ. Empty: a response body and a
	// model field are both full-schema, so every difference found here has been a defect.
	//
	// The `stamp` cell was pinned until the arm learned to follow the declaration's WRITTEN
	// right-hand side. `type Stamp time.Time` is not `time.Time`, so the recognizer keys on identity
	// and declines — but `Underlying()` peeled past the `time.Time` layer entirely, where the
	// recognizer would have seen it, leaving the response to be read as a struct whose fields become
	// headers. time.Time exports none, so the response carried no schema.
	knownBroken := map[string]string{}

	props := doc.Definitions["Host"].Properties
	require.NotEmpty(t, props, "the control host must have properties")

	var ledger strings.Builder
	fmt.Fprintf(&ledger, "\n%-8s %-26s %-26s %s\n", "SUBJECT", "MODEL FIELD", "RESPONSE BODY", "VERDICT")
	fmt.Fprintf(&ledger, "%s\n", strings.Repeat("-", 92))

	for _, c := range []struct{ prop, response string }{
		{"stamp", "stampResp"},
		{"emails", "emailsResp"},
		{"code", "codeResp"},
		{"count", "countResp"},
	} {
		t.Run(c.prop, func(t *testing.T) {
			field, ok := props[c.prop]
			require.True(t, ok, "missing control property %s", c.prop)
			control := schemaSignature(field, doc.Definitions, 0)

			resp, ok := doc.Responses[c.response]
			require.True(t, ok, "missing response %s", c.response)

			body := "<NO SCHEMA>"
			if resp.Schema != nil {
				body = schemaSignature(*resp.Schema, doc.Definitions, 0)
			}

			reason, pinned := knownBroken[c.prop]
			verdict := "OK"
			switch {
			case control == body && pinned:
				verdict = "UNPINNED — remove it from knownBroken"
			case control != body:
				verdict = "PINNED(Q42)"
			}
			fmt.Fprintf(&ledger, "%-8s %-26s %-26s %s\n", c.prop, control, body, verdict)

			if pinned {
				assert.NotEqual(t, control, body,
					"%s: pinned as broken (%s) but the two now agree — remove it", c.prop, reason)

				return
			}
			assert.Equal(t, control, body,
				"a response body must render its declaration as a model field does")
		})
	}

	t.Log(ledger.String())

	// The response types must NOT surface as definitions: a swagger:response declares a response.
	for _, name := range []string{"StampResp", "EmailsResp", "CodeResp", "CountResp"} {
		assert.NotContains(t, doc.Definitions, name,
			"a swagger:response declaration must not publish a definition")
	}

	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_named_nonstruct.json")
}
