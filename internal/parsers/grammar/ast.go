// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"iter"
)

// Block is the interface implemented by every typed AST node the
// parser emits. One Block corresponds to one comment group's parsed
// content. Typed kinds (ModelBlock, RouteBlock, …) embed *baseBlock
// and add the fields specific to their annotation.
//
// See architecture §4.6.
type Block interface {
	// Pos reports the position of the block's defining token — the
	// annotation line for annotated blocks, or the first comment line
	// for UnboundBlock.
	Pos() token.Position
	// Title returns the short one-liner extracted from the comment
	// group (the first non-annotation paragraph, first line).
	Title() string
	// Description returns everything between the title paragraph and
	// the first keyword/block-header, joined by newlines.
	Description() string
	// Diagnostics returns the non-fatal observations accumulated
	// while parsing this block (unknown keywords, context-invalid
	// keywords, malformed values, …).
	Diagnostics() []Diagnostic

	// Properties iterates the keyword:value pairs attached to this
	// block (flat list in source order).
	Properties() iter.Seq[Property]
	// YAMLBlocks iterates the --- fenced YAML bodies captured inside
	// this block (swagger:operation, swagger:meta bodies). The parser
	// does NOT parse YAML; it only isolates the bodies.
	YAMLBlocks() iter.Seq[RawYAML]
	// Extensions iterates the x-* vendor extensions declared under an
	// "extensions:" block inside this Block.
	Extensions() iter.Seq[Extension]

	// Kind returns the top-level annotation kind this Block was
	// dispatched from (UnboundBlock returns AnnUnknown). Used by
	// analyzers to type-switch-check without reflection.
	AnnotationKind() AnnotationKind
}

// AnnotationKind identifies the top-level "swagger:xxx" directive
// that produced this Block. This is distinct from Kind in
// keywords.go (which names the *sub-context* where a keyword may
// appear). AnnotationKind is used for Block dispatch; Kind for
// keyword legality.
type AnnotationKind int

const (
	AnnUnknown AnnotationKind = iota

	AnnRoute       // swagger:route
	AnnOperation   // swagger:operation
	AnnParameters  // swagger:parameters
	AnnResponse    // swagger:response
	AnnModel       // swagger:model
	AnnMeta        // swagger:meta
	AnnStrfmt      // swagger:strfmt
	AnnAlias       // swagger:alias
	AnnName        // swagger:name
	AnnAllOf       // swagger:allOf
	AnnEnumDecl    // swagger:enum
	AnnIgnore      // swagger:ignore
	AnnDefaultName // swagger:default (name override, not the keyword)
	AnnType        // swagger:type
	AnnFile        // swagger:file
)

// Annotation label strings, one per AnnotationKind. Shared by
// AnnotationKind.String() and AnnotationKindFromName() so there is
// exactly one source of truth per label.
const (
	labelRoute      = "route"
	labelOperation  = "operation"
	labelParameters = "parameters"
	labelResponse   = "response"
	labelModel      = "model"
	labelMeta       = "meta"
	labelStrfmt     = "strfmt"
	labelAlias      = "alias"
	labelName       = "name"
	labelAllOf      = "allOf"
	labelEnum       = "enum"
	labelIgnore     = "ignore"
	labelDefault    = "default"
	labelType       = "type"
	labelFile       = "file"
	labelUnknown    = "unknown"
)

// String renders an AnnotationKind as its source label.
func (a AnnotationKind) String() string {
	switch a {
	case AnnRoute:
		return labelRoute
	case AnnOperation:
		return labelOperation
	case AnnParameters:
		return labelParameters
	case AnnResponse:
		return labelResponse
	case AnnModel:
		return labelModel
	case AnnMeta:
		return labelMeta
	case AnnStrfmt:
		return labelStrfmt
	case AnnAlias:
		return labelAlias
	case AnnName:
		return labelName
	case AnnAllOf:
		return labelAllOf
	case AnnEnumDecl:
		return labelEnum
	case AnnIgnore:
		return labelIgnore
	case AnnDefaultName:
		return labelDefault
	case AnnType:
		return labelType
	case AnnFile:
		return labelFile
	case AnnUnknown:
		fallthrough
	default:
		return labelUnknown
	}
}

// AnnotationKindFromName resolves the `swagger:<name>` label (e.g.,
// "model", "route") to the matching AnnotationKind. Returns
// AnnUnknown for names the parser does not recognize at v1 parity.
func AnnotationKindFromName(name string) AnnotationKind {
	switch name {
	case labelRoute:
		return AnnRoute
	case labelOperation:
		return AnnOperation
	case labelParameters:
		return AnnParameters
	case labelResponse:
		return AnnResponse
	case labelModel:
		return AnnModel
	case labelMeta:
		return AnnMeta
	case labelStrfmt:
		return AnnStrfmt
	case labelAlias:
		return AnnAlias
	case labelName:
		return AnnName
	case labelAllOf:
		return AnnAllOf
	case labelEnum:
		return AnnEnumDecl
	case labelIgnore:
		return AnnIgnore
	case labelDefault:
		return AnnDefaultName
	case labelType:
		return AnnType
	case labelFile:
		return AnnFile
	default:
		return AnnUnknown
	}
}

// Property is one keyword:value pair inside a Block's body. Value
// is the raw string as it appeared in the comment; Typed carries the
// primitive-converted form when the keyword's ValueType is one of the
// primitives the parser converts at parse time (Number, Integer,
// Boolean, StringEnum). Raw-typed keywords (RawValue, RawBlock,
// String, CommaList) leave Typed at its zero value and the analyzer
// interprets Value.
type Property struct {
	Keyword    Keyword
	Pos        token.Position
	Value      string
	Typed      TypedValue
	ItemsDepth int // 0 = no "items." nesting
}

// TypedValue carries the primitive-converted form of a keyword's
// value when the keyword's ValueType is Number/Integer/Boolean/
// StringEnum. For other ValueTypes the fields are zero.
//
// Op is the leading comparison operator stripped from a Number value
// ("<", "<=", ">", ">=", "="); empty when no operator was present or
// for non-Number values. v1 accepts e.g. `maximum: <5` to mean
// exclusive-maximum; the analyzer interprets Op + Number to decide
// inclusive vs. exclusive semantics.
type TypedValue struct {
	Type    ValueType
	Op      string
	Number  float64
	Integer int64
	Boolean bool
	String  string // for StringEnum: the canonical (table-spelled) value
}

// RawYAML is one captured YAML body (between --- fences). The parser
// does not parse the content; it records the bytes and the position
// so the analyzer can hand it to internal/parsers/yaml/.
type RawYAML struct {
	Pos  token.Position
	Text string
}

// Extension is one x-* vendor extension entry under an
// "extensions:" block. Value is the raw line content; analyzers parse
// it further if needed (e.g., inline YAML via internal/parsers/yaml/).
type Extension struct {
	Name  string
	Pos   token.Position
	Value string
}

// baseBlock carries the fields common to every Block kind. Typed
// blocks embed *baseBlock and add kind-specific positional data.
type baseBlock struct {
	pos         token.Position
	title       string
	description string
	kind        AnnotationKind

	properties  []Property
	yamlBlocks  []RawYAML
	extensions  []Extension
	diagnostics []Diagnostic
}

func (b *baseBlock) Pos() token.Position            { return b.pos }
func (b *baseBlock) Title() string                  { return b.title }
func (b *baseBlock) Description() string            { return b.description }
func (b *baseBlock) Diagnostics() []Diagnostic      { return b.diagnostics }
func (b *baseBlock) AnnotationKind() AnnotationKind { return b.kind }

func (b *baseBlock) Properties() iter.Seq[Property] {
	return func(yield func(Property) bool) {
		for _, p := range b.properties {
			if !yield(p) {
				return
			}
		}
	}
}

func (b *baseBlock) YAMLBlocks() iter.Seq[RawYAML] {
	return func(yield func(RawYAML) bool) {
		for _, y := range b.yamlBlocks {
			if !yield(y) {
				return
			}
		}
	}
}

func (b *baseBlock) Extensions() iter.Seq[Extension] {
	return func(yield func(Extension) bool) {
		for _, e := range b.extensions {
			if !yield(e) {
				return
			}
		}
	}
}

// --- typed Block kinds ---

// ModelBlock is produced by `swagger:model [Name]`. Name is the
// optional override declared on the annotation line (empty when the
// declaration uses the Go type's own name).
type ModelBlock struct {
	*baseBlock

	Name string
}

// RouteBlock is produced by `swagger:route METHOD /path [tags] opID`.
// All four positional fields are captured by the parser (P1.6).
type RouteBlock struct {
	*baseBlock

	Method string
	Path   string
	Tags   string // free-text tags segment (optional)
	OpID   string
}

// OperationBlock is produced by `swagger:operation METHOD /path [tags] opID`.
// Positional fields match RouteBlock. The YAML body (if any) is in
// baseBlock.yamlBlocks, reachable via YAMLBlocks().
type OperationBlock struct {
	*baseBlock

	Method string
	Path   string
	Tags   string
	OpID   string
}

// ParametersBlock is produced by `swagger:parameters T1 T2 ...`.
// Each positional arg names a Go type whose exported fields become
// the parameter set.
type ParametersBlock struct {
	*baseBlock

	TargetTypes []string
}

// ResponseBlock is produced by `swagger:response [Name]`. Name is the
// optional override for how the response is exposed in the spec.
type ResponseBlock struct {
	*baseBlock

	Name string
}

// MetaBlock is produced by `swagger:meta`. The info-block keywords
// (version, host, basePath, license, contact, …) appear as
// Properties; YAML bodies in YAMLBlocks.
type MetaBlock struct {
	*baseBlock
}

// UnboundBlock represents a comment group with no annotation line —
// e.g., a struct field's docstring carrying validations. AnnotationKind
// returns AnnUnknown; analyzers interpret Properties based on the
// enclosing Go declaration (scanner context).
type UnboundBlock struct {
	*baseBlock
}

// --- constructors for the parser (P1.4) ---

// newBaseBlock initializes a baseBlock for the given annotation kind
// at the given source position. Returns a pointer so typed Block
// kinds can embed it and share the state.
func newBaseBlock(kind AnnotationKind, pos token.Position) *baseBlock {
	return &baseBlock{pos: pos, kind: kind}
}
