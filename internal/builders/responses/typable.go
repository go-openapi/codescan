// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package responses

import (
	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/builders/schema"
	"github.com/go-openapi/codescan/internal/ifaces"
	oaispec "github.com/go-openapi/spec"
)

type responseTypable struct {
	in       string
	header   *oaispec.Header
	response *oaispec.Response
	skipExt  bool

	// refAttempted: caller-owned flag flipped when SetRef is called
	// under non-body mode. See [§typable](./README.md#typable).
	refAttempted *bool
}

func (ht responseTypable) In() string { return ht.in }

func (ht responseTypable) Level() int { return 0 }

// Typed writes the primitive type onto the body schema in body mode,
// or onto the header in SimpleSchema mode. Without the body branch a
// primitive `Body` field (e.g. `Body string`) lands its type on the
// header, which body responses discard — leaving the response with no
// schema at all (go-swagger#2942). Mirrors SetRef's body/non-body
// split.
func (ht responseTypable) Typed(tpe, format string) {
	if ht.in == inBody {
		ht.Schema().Typed(tpe, format)
		return
	}
	ht.header.Typed(tpe, format)
}

func (ht responseTypable) Items() ifaces.SwaggerTypable { //nolint:ireturn // polymorphic by design
	bdt, schema := schema.BodyTypable(ht.in, ht.response.Schema, ht.skipExt)
	if bdt != nil {
		ht.response.Schema = schema
		return bdt
	}

	if ht.header.Items == nil {
		ht.header.Items = new(oaispec.Items)
	}

	ht.header.Type = "array"

	return resolvers.NewItemsTypable(ht.header.Items, 1, "header")
}

// SetRef writes the ref onto the body schema in body mode; under
// non-body it no-ops and flips refAttempted (Q2). See
// [§typable](./README.md#typable).
func (ht responseTypable) SetRef(ref oaispec.Ref) {
	if ht.in == inBody {
		ht.Schema().Ref = ref
		return
	}
	if ht.refAttempted != nil {
		*ht.refAttempted = true
	}
}

func (ht responseTypable) Schema() *oaispec.Schema {
	if ht.response.Schema == nil {
		ht.response.Schema = new(oaispec.Schema)
	}

	return ht.response.Schema
}

func (ht responseTypable) AddExtension(key string, value any) {
	ht.response.AddExtension(key, value)
}

func (ht responseTypable) WithEnum(values ...any) {
	// Spread the variadic through: passing the slice itself would nest it
	// one level deep (enum: [[FIRST, SECOND]]), producing malformed OAS2.
	// Mirrors paramTypable / schema.Typable / ItemsTypable.
	ht.header.WithEnum(values...)
}

func (ht responseTypable) WithEnumDescription(_ string) {
	// Intentionally a no-op. A response header is an OAS v2 SimpleSchema
	// target, but go-openapi/spec's Header.MarshalJSON serializes only
	// CommonValidations + SimpleSchema + HeaderProps — it does NOT emit the
	// embedded VendorExtensible. So a header cannot carry x-go-enum-desc (or
	// any x-go-* extension) in the output spec; setting one would be dead
	// in-memory state silently dropped at marshal time. The enum *values*
	// still ship via WithEnum. This is the one SimpleSchema target where the
	// const-name mapping is unrepresentable. See go-swagger/go-swagger#2922.
	//
	// TODO: tracked by go-openapi/spec#277 (Header.MarshalJSON drops
	// VendorExtensible). Once that lands and the dependency is bumped, wire
	// this to emit x-go-enum-desc (mirroring paramTypable.WithEnumDescription)
	// and flip the header assertion in coverage_bug_2922_test.go from
	// "no x-go-enum-desc" to "extension present".
}

// SimpleSchemaShape satisfies schema.SimpleSchemaProbe (non-body
// path; body uses WithType). See [§typable](./README.md#typable).
func (ht responseTypable) SimpleSchemaShape() *oaispec.SimpleSchema {
	return &ht.header.SimpleSchema
}

// HasRef satisfies schema.SimpleSchemaProbe. True when a non-body
// SetRef attempt was recorded — the exit validator emits
// CodeUnsupportedInSimpleSchema. See [§typable](./README.md#typable).
func (ht responseTypable) HasRef() bool {
	return ht.refAttempted != nil && *ht.refAttempted
}

// ResetForViolation satisfies schema.SimpleSchemaProbe. Wipes the
// header's SimpleSchema back to `{}`.
func (ht responseTypable) ResetForViolation() {
	ht.header.SimpleSchema = oaispec.SimpleSchema{}
}
