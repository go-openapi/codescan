// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2637 locks the fix for go-swagger issue #2637: a local type defined from a
// same-named type in another package (`type CreateDomainRequest mongo.CreateDomainRequest`) used to
// collide on the short key and emit a definition whose body was a `$ref` TO ITSELF — invalid OAS
// that hangs downstream codegen.
//
// Now the local type and mongo's get distinct concat-qualified names, so the local definition's
// body is a `$ref` to the MONGO definition — a valid cross-type reference, no self-`$ref`.
// Same family as #2783..
func TestCoverage_Bug2637(t *testing.T) {
	doc, diags := nameIdentityDocDiags(t, "./bugs/2637/...")

	require.Len(t, doc.Definitions, 2, "the local and the mongo CreateDomainRequest are distinct")

	// The local defined type (package bug2637, dir "2637") qualifies to X2637CreateDomainRequest; the
	// mongo one to MongoCreateDomainRequest.
	local, ok := doc.Definitions["X2637CreateDomainRequest"]
	require.True(t, ok, "the local defined type keeps its own (qualified) definition")
	require.Contains(t, doc.Definitions, "MongoCreateDomainRequest")

	ref := local.Ref.String()
	assert.NotEqual(t, "#/definitions/X2637CreateDomainRequest", ref,
		"G2: the definition must not reference itself")
	assert.Equal(t, "#/definitions/MongoCreateDomainRequest", ref,
		"the local defined type resolves to the mongo definition")
	assert.True(t, hasDiagnostic(diags, grammar.CodeCollidingModelName))

	scantest.CompareOrDumpJSON(t, doc, "bugs_2637_schema.json")
}
