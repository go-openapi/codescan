// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammartest

import (
	"path/filepath"
	"testing"
)

// Harness smoke tests: pick a handful of representative comment-group
// shapes and lock their v2 parse-output down as golden JSON. P5
// builder-migration commits extend the fixture set per builder.
//
// Fixture sources stay inline in these tests for the moment — they
// read like a readable "what the v2 parser produces for THIS comment"
// catalogue. Migration to external Go-package fixtures (the
// fixtures/ tree) happens when P5 needs to cover full-file scenarios.

func TestHarnessSimpleModel(t *testing.T) {
	src := `package p

// swagger:model Foo
//
// Foo is a simple model.
//
// maximum: 100
// minimum: 0
// pattern: ^[a-z]+$
type Foo int
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "simple_model.json"), views)
}

func TestHarnessRouteWithTags(t *testing.T) {
	src := `package p

// swagger:route GET /pets tags listPets
//
// consumes:
//   application/json
// produces:
//   application/json
func ListPets() {}
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "route_with_tags.json"), views)
}

func TestHarnessOperationWithYAML(t *testing.T) {
	src := `package p

// swagger:operation GET /pets listPets
//
// ---
// responses:
//   200: successResponse
//   404: notFound
// ---
func ListPets() {}
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "operation_with_yaml.json"), views)
}

func TestHarnessParametersWithValidations(t *testing.T) {
	src := `package p

// swagger:parameters listPets
//
// in: query
// required: true
// maximum: 100
// minimum: 0
type PetParams struct{}
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "parameters_with_validations.json"), views)
}

func TestHarnessMetaWithExtensions(t *testing.T) {
	src := `package p

// swagger:meta
//
// version: "1.0"
// host: api.example.com
//
// extensions:
//   x-foo: bar
//   x-baz: 42
type Root struct{}
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "meta_with_extensions.json"), views)
}

func TestHarnessUnboundWithBullet(t *testing.T) {
	// Regression: bullet dashes survive (P1.10 lock-in) and no
	// annotation → UnboundBlock path.
	src := `package p

// A summary line.
//
// - first bullet
// - second bullet
type Foo int
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "unbound_with_bullet.json"), views)
}

func TestHarnessContextInvalidDiagnostic(t *testing.T) {
	// Regression: context-validity warning surfaces in the view via
	// the normalized diagnostics channel.
	src := `package p

// swagger:model Foo
// in: query
type Foo int
`
	views := ParseSourceToViews(t, src)
	AssertGoldenView(t, filepath.Join("testdata", "golden", "context_invalid_diag.json"), views)
}
