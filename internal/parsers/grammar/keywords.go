// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import "strings"

// Lookup returns the Keyword matching the given name (or alias),
// case-insensitively. The second return value reports whether a match
// was found.
//
// The lexer (P1.2) uses this to classify `keyword:` lines. It operates
// over the table in keywords_table.go.
func Lookup(name string) (Keyword, bool) {
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return Keyword{}, false
	}
	for _, kw := range keywords {
		if strings.EqualFold(kw.Name, needle) {
			return kw, true
		}
		for _, alias := range kw.Aliases {
			if strings.EqualFold(alias, needle) {
				return kw, true
			}
		}
	}
	return Keyword{}, false
}

// Keywords returns the authoritative keyword table.
//
// The docs generator (gen/main.go) consumes it to emit
// docs/annotation-keywords.md; tooling that introspects the keyword set
// (LSP, `codescan grammar check`) reads the same slice.
func Keywords() []Keyword {
	out := make([]Keyword, len(keywords))
	copy(out, keywords)
	return out
}

// Kind identifies a context where `keyword: value` pairs may appear.
//
// These are the sub-scopes within an annotation block (finer-grained than
// the top-level swagger:xxx kind, which lives on the Block family in
// ast.go). The parser is context-free: it recognizes every keyword
// regardless of enclosing annotation, and uses this per-keyword legality
// set to emit non-fatal `parse.context-invalid` diagnostics.
//
// LSP completion consults the same data to filter suggestions.
type Kind int

const (
	KindUnknown Kind = iota

	KindParam     // an individual parameter (field under swagger:parameters)
	KindHeader    // a response header entry
	KindSchema    // a schema property (model field or definition)
	KindItems     // a nested array-items subscope
	KindRoute     // a metadata line under swagger:route
	KindOperation // a metadata line under swagger:operation (non-YAML body)
	KindMeta      // a metadata line under swagger:meta
	KindResponse  // a response-level property (status, description, …)
)

// String renders a Kind as the lowercase label used in diagnostics and
// docs.
func (k Kind) String() string {
	switch k {
	case KindParam:
		return "param"
	case KindHeader:
		return "header"
	case KindSchema:
		return "schema"
	case KindItems:
		return "items"
	// The string literals below intentionally duplicate some of the
	// labelXxx constants defined in ast.go. Kind (keyword context) and
	// AnnotationKind (Block dispatch) are separate concerns that happen
	// to share a handful of label spellings; see architecture §4.6 —
	// coupling them through a shared const would hide that distinction.
	//
	//nolint:goconst // see note above
	case KindRoute:
		return "route"
	case KindOperation:
		return "operation"
	case KindMeta:
		return "meta"
	case KindResponse:
		return "response"
	case KindUnknown:
		fallthrough
	default:
		return "unknown"
	}
}

// ValueType categorizes the expected shape of a keyword's value. Primitive
// types (Number, Integer, Boolean, StringEnum) are converted inside the
// parser at parse time; RawValue defers type-conversion to the analyzer
// when the target Go type determines it (e.g., `default:`, `example:`).
type ValueType int

const (
	ValueNone ValueType = iota

	ValueNumber     // decimal; e.g. maximum, minimum, multipleOf
	ValueInteger    // unsigned count; e.g. maxLength, minItems
	ValueBoolean    // true/false; e.g. required, readOnly
	ValueString     // opaque string; e.g. pattern (verbatim, not interpreted)
	ValueCommaList  // comma-separated values; e.g. enum, schemes
	ValueStringEnum // one of a fixed set; e.g. in, collectionFormat
	ValueRawBlock   // multi-line block body (headers like consumes:, security:)
	ValueRawValue   // raw string; analyzer type-converts per field Go type
)

// String renders a ValueType as the label used in diagnostics and docs.
func (v ValueType) String() string {
	switch v {
	case ValueNumber:
		return "number"
	case ValueInteger:
		return "integer"
	case ValueBoolean:
		return "boolean"
	case ValueString:
		return "string"
	case ValueCommaList:
		return "comma-list"
	case ValueStringEnum:
		return "string-enum"
	case ValueRawBlock:
		return "raw-block"
	case ValueRawValue:
		return "raw-value"
	case ValueNone:
		fallthrough
	default:
		return "none"
	}
}

// Keyword describes one recognizable `keyword: value` form.
type Keyword struct {
	Name     string
	Aliases  []string
	Value    Value
	Contexts []ContextDoc
	Doc      string
}

// Value captures the expected value shape plus any fixed enumeration.
type Value struct {
	Type   ValueType
	Values []string // non-nil when Type == ValueStringEnum
}

// ContextDoc binds a legal context to a context-specific doc string.
// Separate docs per Kind let LSP show tooltips tailored to where the
// cursor is (a `maximum` in a parameter vs. a `maximum` in a schema).
type ContextDoc struct {
	Kind Kind
	Doc  string
}

// keyword constructs a Keyword from functional options. Used by the
// table in keywords_table.go.
func keyword(name string, opts ...keywordOpt) Keyword {
	kw := Keyword{Name: name}
	for _, o := range opts {
		o(&kw)
	}
	return kw
}

type keywordOpt func(*Keyword)

// --- value-type options ---

func asNumber() keywordOpt    { return func(kw *Keyword) { kw.Value.Type = ValueNumber } }
func asInteger() keywordOpt   { return func(kw *Keyword) { kw.Value.Type = ValueInteger } }
func asBoolean() keywordOpt   { return func(kw *Keyword) { kw.Value.Type = ValueBoolean } }
func asString() keywordOpt    { return func(kw *Keyword) { kw.Value.Type = ValueString } }
func asCommaList() keywordOpt { return func(kw *Keyword) { kw.Value.Type = ValueCommaList } }
func asRawBlock() keywordOpt  { return func(kw *Keyword) { kw.Value.Type = ValueRawBlock } }
func asRawValue() keywordOpt  { return func(kw *Keyword) { kw.Value.Type = ValueRawValue } }

func asStringEnum(values ...string) keywordOpt {
	return func(kw *Keyword) {
		kw.Value.Type = ValueStringEnum
		kw.Value.Values = values
	}
}

// --- alias option ---

func aka(names ...string) keywordOpt {
	return func(kw *Keyword) { kw.Aliases = append(kw.Aliases, names...) }
}

// --- per-context legality + doc (Option A) ---

func inParam(doc string) keywordOpt     { return legalIn(KindParam, doc) }
func inHeader(doc string) keywordOpt    { return legalIn(KindHeader, doc) }
func inSchema(doc string) keywordOpt    { return legalIn(KindSchema, doc) }
func inItems(doc string) keywordOpt     { return legalIn(KindItems, doc) }
func inRoute(doc string) keywordOpt     { return legalIn(KindRoute, doc) }
func inOperation(doc string) keywordOpt { return legalIn(KindOperation, doc) }
func inMeta(doc string) keywordOpt      { return legalIn(KindMeta, doc) }

func legalIn(kind Kind, doc string) keywordOpt {
	return func(kw *Keyword) {
		kw.Contexts = append(kw.Contexts, ContextDoc{Kind: kind, Doc: doc})
	}
}
