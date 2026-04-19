// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"testing"

	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/spec"

	"github.com/go-openapi/testify/v2/assert"
)

func TestOperationsParser(t *testing.T) {
	sctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{
			"./goparsing/classification",
			"./goparsing/classification/models",
			"./goparsing/classification/operations",
			"./goparsing/classification/operations_annotation",
		},
		WorkDir: scantest.FixturesDir(),
	})
	require.NoError(t, err)
	var ops spec.Paths
	for apiPath := range sctx.Operations() {
		prs := &Builder{
			ctx:        sctx,
			path:       apiPath,
			operations: make(map[string]*spec.Operation),
		}
		require.NoError(t, prs.Build(&ops))
	}

	assert.Len(t, ops.Paths, 3)

	po, ok := ops.Paths["/pets"]
	assert.TrueT(t, ok)
	require.NotNil(t, po.Get)
	assertAnnotationOperation(t,
		po.Get,
		"getPet",
		"",
		"List all pets",
		[]string{"pets"},
	)
	if po.Get != nil {
		rsp, k := po.Get.Responses.StatusCodeResponses[200]
		if assert.TrueT(t, k) {
			assert.EqualT(t, "An paged array of pets", rsp.Description)
		}
		if assert.NotNil(t, po.Get.Responses.Default) {
			assert.EqualT(t, "unexpected error", po.Get.Responses.Default.Description)
		}
	}

	po, ok = ops.Paths["/pets/{id}"]
	assert.TrueT(t, ok)
	assert.NotNil(t, po.Put)
	assertAnnotationOperation(t,
		po.Put,
		"updatePet",
		"Updates the details for a pet.",
		"Some long explanation,\nspanning over multipele lines,\nAKA the description.",
		[]string{"pets"},
	)
	if po.Put != nil {
		rsp, k := po.Put.Responses.StatusCodeResponses[400]
		if assert.TrueT(t, k) {
			assert.EqualT(t, "Invalid ID supplied", rsp.Description)
		}
		rsp, k = po.Put.Responses.StatusCodeResponses[404]
		if assert.TrueT(t, k) {
			assert.EqualT(t, "Pet not found", rsp.Description)
		}
		rsp, k = po.Put.Responses.StatusCodeResponses[405]
		if assert.TrueT(t, k) {
			assert.EqualT(t, "Validation exception", rsp.Description)
		}
	}

	po, ok = ops.Paths["/v1/events"]
	assert.TrueT(t, ok)
	assert.NotNil(t, po.Get)
	assertAnnotationOperation(t,
		po.Get,
		"getEvents",
		"Events",
		"Mitigation Events",
		[]string{"Events"},
	)
	if po.Get != nil {
		rsp, k := po.Get.Responses.StatusCodeResponses[200]
		if assert.TrueT(t, k) {
			assert.EqualT(t, "#/definitions/ListResponse", rsp.Schema.Ref.String())
			assert.EqualT(t, "200", rsp.Description)
		}
		rsp, k = po.Get.Responses.StatusCodeResponses[400]
		if assert.TrueT(t, k) {
			assert.EqualT(t, "#/definitions/ErrorResponse", rsp.Schema.Ref.String())
			assert.EqualT(t, "400", rsp.Description)
		}
	}

	scantest.CompareOrDumpJSON(t, ops, "classification_operations.json")
}

func assertAnnotationOperation(t *testing.T, op *spec.Operation, id, summary, description string, tags []string) {
	assert.NotNil(t, op)
	assert.EqualT(t, summary, op.Summary)
	assert.EqualT(t, description, op.Description)
	assert.EqualT(t, id, op.ID)
	assert.Equal(t, tags, op.Tags)
	assert.SliceContainsT(t, op.Consumes, "application/json")
	assert.SliceContainsT(t, op.Consumes, "application/xml")
	assert.SliceContainsT(t, op.Produces, "application/json")
	assert.SliceContainsT(t, op.Produces, "application/xml")
	assert.Len(t, op.Security, 1)
	if len(op.Security) > 0 {
		akv, ok := op.Security[0]["petstore_auth"]
		assert.TrueT(t, ok)
		// akv must be defined & not empty
		assert.NotNil(t, akv)
		assert.NotEmpty(t, akv)
	}
}
