// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"regexp"
	"strings"

	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

var _ ifaces.ValidationBuilder = &SchemaValidations{}

// SchemaOptions configures the full-Schema dispatch.
//
// SimpleSchemaMode enables the SimpleSchema gating used by the
// parameter/header drivers that pass through the schema builder:
// full-Schema-only keywords (readOnly, discriminator, ...) emit
// CodeUnsupportedInSimpleSchema and skip the write; `required:` is
// silently skipped.
//
// See [§simple-schema-keywords](./README.md#simple-schema-keywords).
type SchemaOptions struct {
	SimpleSchemaMode bool
}

// SchemaValidations adapts *oaispec.Schema to ifaces.ValidationBuilder
// so the full-Schema handler family can write level-0 and items-level
// validations through a uniform target.
//
// Exported because the schema package's refOverrideCollector also uses
// it directly when building $ref-override allOf compounds.
type SchemaValidations struct {
	current *oaispec.Schema
}

// NewSchemaValidations builds an adapter around ps.
func NewSchemaValidations(ps *oaispec.Schema) SchemaValidations {
	return SchemaValidations{current: ps}
}

func (sv SchemaValidations) SetMaximum(val float64, exclusive bool) {
	sv.current.Maximum = &val
	sv.current.ExclusiveMaximum = exclusive
}

func (sv SchemaValidations) SetMinimum(val float64, exclusive bool) {
	sv.current.Minimum = &val
	sv.current.ExclusiveMinimum = exclusive
}
func (sv SchemaValidations) SetMultipleOf(val float64) { sv.current.MultipleOf = &val }
func (sv SchemaValidations) SetMinItems(val int64)     { sv.current.MinItems = &val }
func (sv SchemaValidations) SetMaxItems(val int64)     { sv.current.MaxItems = &val }
func (sv SchemaValidations) SetMinLength(val int64)    { sv.current.MinLength = &val }
func (sv SchemaValidations) SetMaxLength(val int64)    { sv.current.MaxLength = &val }
func (sv SchemaValidations) SetPattern(val string)     { sv.current.Pattern = val }
func (sv SchemaValidations) SetUnique(val bool)        { sv.current.UniqueItems = val }
func (sv SchemaValidations) SetDefault(val any)        { sv.current.Default = val }
func (sv SchemaValidations) SetExample(val any)        { sv.current.Example = val }

// SetEnum writes the parsed enum onto the schema and strips any
// inherited x-go-enum-desc that the field-level override has made
// stale — see [§stale-enum-desc](./README.md#stale-enum-desc).
func (sv SchemaValidations) SetEnum(val string) {
	var typ string
	if len(sv.current.Type) > 0 {
		typ = sv.current.Type[0]
	}
	sv.current.Enum = validations.ParseEnumValues(val, typ, sv.current.Format)
	clearStaleEnumDesc(sv.current)
}

// DispatchSchemaLevel0 routes every level-0 Property in block into
// ps. The enclosing schema receives required/discriminator writes
// keyed by name (name == "" silently skips those cross-target writes,
// for the top-level model case where there is no enclosing).
//
// opts.SimpleSchemaMode gates full-Schema-only keywords with
// CodeUnsupportedInSimpleSchema (for the parameters/headers paths
// that drive the schema builder under SimpleSchema constraints).
//
// diag may be nil; when nil, all diagnostics are dropped.
func DispatchSchemaLevel0(block grammar.Block, enclosing, ps *oaispec.Schema, name string,
	diag func(grammar.Diagnostic), opts SchemaOptions,
) {
	valid := NewSchemaValidations(ps)

	block.Walk(grammar.Walker{
		FilterDepth: 0,
		Number:      schemaNumberHandler(ps, valid, diag),
		Integer:     schemaIntegerHandler(ps, valid, diag),
		Bool:        schemaBoolHandler(enclosing, ps, name, valid, diag, opts),
		String:      schemaStringHandler(ps, valid, diag),
		Raw:         schemaRawHandler(ps, valid),
		Extension:   Extension(ps),
		Diagnostic:  diag,
	})
}

// DispatchSchemaItemsLevel dispatches items-depth Property entries
// onto target. Items elements don't carry vendor extensions and don't
// participate in cross-target writes (required/discriminator), so the
// dispatcher is narrower than the level-0 entry — only number/
// integer/bool(unique)/string/raw handlers fire.
//
// opts.SimpleSchemaMode is accepted for symmetry but currently
// doesn't alter items-level behaviour.
//
// diag may be nil; when nil, all diagnostics are dropped.
func DispatchSchemaItemsLevel(block grammar.Block, target *oaispec.Schema, depth int,
	diag func(grammar.Diagnostic), _ SchemaOptions,
) {
	valid := NewSchemaValidations(target)

	block.Walk(grammar.Walker{
		FilterDepth: depth,
		Number:      schemaNumberHandler(target, valid, diag),
		Integer:     schemaIntegerHandler(target, valid, diag),
		Bool: func(p grammar.Property, val bool) {
			if !p.IsTyped() {
				return
			}
			if !checkShape(p, target, diag) {
				return
			}
			if p.Keyword.Name == grammar.KwUnique {
				valid.SetUnique(val)
			}
		},
		String:     schemaStringHandler(target, valid, diag),
		Raw:        schemaRawHandler(target, valid),
		Diagnostic: diag,
	})
}

// ApplyPattern stores a regex pattern on valid and runs a best-effort
// RE2 hygiene check, emitting CodeInvalidAnnotation when the regex
// does not compile in Go's RE2 engine. The pattern is kept on the
// schema regardless — downstream tools may use JSON Schema's wider
// regex dialect.
//
// Exported because the schema package's refOverrideCollector applies
// the same pattern semantics when building $ref-override allOf
// compounds.
//
// diag may be nil; when nil, the RE2 mismatch warning is dropped.
func ApplyPattern(p grammar.Property, valid SchemaValidations, val string, diag func(grammar.Diagnostic)) {
	valid.SetPattern(val)
	if val == "" {
		return
	}
	if _, err := regexp.Compile(val); err != nil && diag != nil {
		diag(grammar.Diagnostic{
			Pos:      p.Pos,
			Severity: grammar.SeverityWarning,
			Code:     grammar.CodeInvalidAnnotation,
			Message: strings.TrimSpace(
				"pattern: " + val + " is not a valid Go RE2 regex (" + err.Error() + "); " +
					"the value is preserved on the schema but downstream RE2 validators will fail",
			),
		})
	}
}

// SetRequired adds or removes name from the enclosing schema's Required slice.
//
// Exported because the schema package's refOverrideCollector reuses this when handling `required:`
// on a $ref-override field.
func SetRequired(enclosing *oaispec.Schema, name string, required bool) {
	if enclosing == nil {
		return
	}
	midx := -1
	for i, nm := range enclosing.Required {
		if nm == name {
			midx = i
			break
		}
	}
	if required {
		if midx < 0 {
			enclosing.Required = append(enclosing.Required, name)
		}
		return
	}
	if midx >= 0 {
		enclosing.Required = append(enclosing.Required[:midx], enclosing.Required[midx+1:]...)
	}
}

// SetDiscriminator writes name to enclosing.Discriminator when
// required=true, or clears it when required=false and the current
// value matches.
func SetDiscriminator(enclosing *oaispec.Schema, name string, required bool) {
	if enclosing == nil {
		return
	}
	if required {
		enclosing.Discriminator = name
		return
	}
	if enclosing.Discriminator == name {
		enclosing.Discriminator = ""
	}
}

// SchemaTypeOf returns the resolved Swagger type of ps, or "" when
// the schema has no Type set (typically a $ref-only schema, where
// type-aware coercion is not possible at this level).
func SchemaTypeOf(ps *oaispec.Schema) string {
	if ps == nil || len(ps.Type) == 0 {
		return ""
	}
	return ps.Type[0]
}

// --- internal per-shape handler factories ---

// schemaNumberHandler returns a Walker.Number callback bound to valid.
// Recognises maximum / minimum / multipleOf. Skips on typing failure
// (parser already emitted a CodeInvalidNumber) or shape mismatch
// (checkShape emits CodeShapeMismatch and the property is dropped).
func schemaNumberHandler(ps *oaispec.Schema, valid SchemaValidations,
	diag func(grammar.Diagnostic),
) func(grammar.Property, float64, bool) {
	return func(p grammar.Property, val float64, exclusive bool) {
		if !p.IsTyped() {
			return
		}
		if !checkShape(p, ps, diag) {
			return
		}
		switch p.Keyword.Name {
		case grammar.KwMaximum:
			valid.SetMaximum(val, exclusive)
		case grammar.KwMinimum:
			valid.SetMinimum(val, exclusive)
		case grammar.KwMultipleOf:
			valid.SetMultipleOf(val)
		}
	}
}

// schemaIntegerHandler returns a Walker.Integer callback bound to
// valid. Recognises minLength / maxLength / minItems / maxItems.
// Skips on typing failure or shape mismatch.
func schemaIntegerHandler(ps *oaispec.Schema, valid SchemaValidations,
	diag func(grammar.Diagnostic),
) func(grammar.Property, int64) {
	return func(p grammar.Property, val int64) {
		if !p.IsTyped() {
			return
		}
		if !checkShape(p, ps, diag) {
			return
		}
		switch p.Keyword.Name {
		case grammar.KwMinLength:
			valid.SetMinLength(val)
		case grammar.KwMaxLength:
			valid.SetMaxLength(val)
		case grammar.KwMinItems:
			valid.SetMinItems(val)
		case grammar.KwMaxItems:
			valid.SetMaxItems(val)
		}
	}
}

// schemaBoolHandler returns a Walker.Bool callback. Required and
// discriminator route to the enclosing schema keyed by name;
// unique/readOnly route to the property schema.
//
// Under SimpleSchemaMode, full-Schema-only keywords (readOnly,
// discriminator) emit CodeUnsupportedInSimpleSchema and skip;
// `required:` is silently skipped — see
// [§simple-schema-keywords](./README.md#simple-schema-keywords).
func schemaBoolHandler(enclosing, ps *oaispec.Schema, name string, valid SchemaValidations,
	diag func(grammar.Diagnostic), opts SchemaOptions,
) func(grammar.Property, bool) {
	return func(p grammar.Property, val bool) {
		if !p.IsTyped() {
			return
		}
		if !checkShape(p, ps, diag) {
			return
		}
		if opts.SimpleSchemaMode && !IsSimpleSchemaKeyword(p.Keyword.Name) {
			if diag != nil {
				diag(grammar.Warnf(
					p.Pos,
					grammar.CodeUnsupportedInSimpleSchema,
					"%q is a full-Schema-only keyword and is not allowed under SimpleSchema mode; ignored",
					p.Keyword.Name,
				))
			}
			return
		}
		if opts.SimpleSchemaMode && p.Keyword.Name == grammar.KwRequired {
			return
		}
		switch p.Keyword.Name {
		case grammar.KwUnique:
			valid.SetUnique(val)
		case grammar.KwReadOnly:
			ps.ReadOnly = val
		case grammar.KwRequired:
			if name != "" {
				SetRequired(enclosing, name, val)
			}
		case grammar.KwDiscriminator:
			if name != "" {
				SetDiscriminator(enclosing, name, val)
			}
		}
	}
}

// schemaStringHandler returns a Walker.String callback.
//
// Recognises pattern (raw regex) and the enum keyword's pre-typed enum-option
// form (rare; comma-list / JSON-array forms travel through Raw).
//
// Shape-checks pattern (string-only); default/example/enum are
// type-independent (ParseDefault handles the type-specific coercion).
func schemaStringHandler(ps *oaispec.Schema, valid SchemaValidations,
	diag func(grammar.Diagnostic),
) func(grammar.Property, string) {
	return func(p grammar.Property, val string) {
		if !checkShape(p, ps, diag) {
			return
		}
		switch p.Keyword.Name {
		case grammar.KwPattern:
			ApplyPattern(p, valid, val, diag)
		case grammar.KwDefault:
			if v, err := validations.ParseDefault(val, SchemaTypeOf(ps), ps.Format); err == nil {
				valid.SetDefault(v)
			}
		case grammar.KwExample:
			if v, err := validations.ParseDefault(val, SchemaTypeOf(ps), ps.Format); err == nil {
				valid.SetExample(v)
			}
		case grammar.KwEnum:
			valid.SetEnum(val)
		}
	}
}

// schemaRawHandler routes ShapeRawBlock / ShapeRawValue / ShapeNone /
// ShapeCommaList properties — default:, example:, enum: when
// expressed as raw bodies. Extensions travel through Walker.Extension
// with YAML-typed values, so the KwExtensions arm is absent here.
func schemaRawHandler(ps *oaispec.Schema, valid SchemaValidations) func(grammar.Property) {
	return func(p grammar.Property) {
		switch p.Keyword.Name {
		case grammar.KwDefault:
			if v, err := validations.ParseDefault(p.Value, SchemaTypeOf(ps), ps.Format); err == nil {
				valid.SetDefault(v)
			}
		case grammar.KwExample:
			if v, err := validations.ParseDefault(p.Value, SchemaTypeOf(ps), ps.Format); err == nil {
				valid.SetExample(v)
			}
		case grammar.KwEnum:
			valid.SetEnum(p.Value)
		}
	}
}

// checkShape gates a Walker callback on
// validations.IsLegalForType(p.Keyword, schema-type). On mismatch,
// emits a CodeShapeMismatch diagnostic and returns false so the
// caller drops the property.
func checkShape(p grammar.Property, ps *oaispec.Schema, diag func(grammar.Diagnostic)) bool {
	var schemaType string
	if ps != nil && len(ps.Type) > 0 {
		schemaType = ps.Type[0]
	}
	ok, hint := validations.IsLegalForType(p.Keyword, schemaType)
	if ok {
		return true
	}
	if diag != nil {
		diag(grammar.Diagnostic{
			Pos:      p.Pos,
			Severity: grammar.SeverityWarning,
			Code:     grammar.CodeShapeMismatch,
			Message:  hint,
		})
	}
	return false
}

// clearStaleEnumDesc removes the x-go-enum-desc extension and strips
// the matching suffix from ps.Description.
//
// Called from SetEnum when a field-level `enum:` overrides a
// type-level `swagger:enum` — see
// [§stale-enum-desc](./README.md#stale-enum-desc).
func clearStaleEnumDesc(ps *oaispec.Schema) {
	enumDesc := resolvers.GetEnumDesc(ps.Extensions)
	if enumDesc == "" {
		return
	}
	delete(ps.Extensions, resolvers.ExtEnumDesc)
	ps.Description = strings.TrimSuffix(
		strings.TrimSuffix(ps.Description, enumDesc),
		"\n",
	)
}
