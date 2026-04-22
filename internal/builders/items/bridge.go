// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package items

import (
	"strings"

	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// ApplyBlock writes every items-level validation Property from b to
// target, filtered to the given nesting depth. This is the v2
// bridge-tagger replacement for the regex-based itemsTaggers() /
// SectionedParser combo — a pure interface swap with no behavior
// change: the ValidationBuilder methods it calls are the same
// targets the v1 taggers wrote through.
//
// `level` is 1-indexed to match v1's rxItemsPrefixFmt semantics:
// level 1 consumes properties whose grammar-parser ItemsDepth is 1
// (e.g., `items.maximum: 5`); level 2 consumes depth-2 properties
// (`items.items.maximum: 5`); and so on. Properties at other depths
// are ignored by this call — the schema-side caller recurses with
// `level+1` for each nested array layer.
//
// Enum / default / example are delegated to target.SetEnum /
// SetDefault / SetExample which route through v1's ParseEnum or
// raw-value storage; this preserves parity end-to-end. The
// eventual swap to internal/parsers/enum.Parse happens in a
// post-migration cleanup commit where v2-only semantics take
// over.
//
// See:
//   - .claude/plans/p5.1a-items-walkthrough.md (design trace)
//   - .claude/plans/p5-builder-migrations.md §4.2 (items scope)
//   - legacy-stop-points.md (bridge-tagger obligations around
//     implied stops; items has no block-head keywords, so S6 is
//     not applicable here).
func ApplyBlock(b grammar.Block, target ifaces.ValidationBuilder, level int) {
	for p := range b.Properties() {
		if p.ItemsDepth != level {
			continue
		}
		dispatchItemsKeyword(p, target)
	}
}

// dispatchItemsKeyword routes one Property to the matching
// ValidationBuilder method. Non-convertible Typed values (where
// the parser's primitive-typing failed and emitted a diagnostic
// upstream) are silently skipped — mirrors v1's tagger behavior
// of early-return on regex match failure.
func dispatchItemsKeyword(p grammar.Property, t ifaces.ValidationBuilder) {
	switch p.Keyword.Name {
	case "maximum":
		if p.Typed.Type == grammar.ValueNumber {
			t.SetMaximum(p.Typed.Number, p.Typed.Op == "<")
		}
	case "minimum":
		if p.Typed.Type == grammar.ValueNumber {
			t.SetMinimum(p.Typed.Number, p.Typed.Op == ">")
		}
	case "multipleOf":
		if p.Typed.Type == grammar.ValueNumber {
			t.SetMultipleOf(p.Typed.Number)
		}
	case "minLength":
		if p.Typed.Type == grammar.ValueInteger {
			t.SetMinLength(p.Typed.Integer)
		}
	case "maxLength":
		if p.Typed.Type == grammar.ValueInteger {
			t.SetMaxLength(p.Typed.Integer)
		}
	case "pattern":
		t.SetPattern(p.Value)
	case "minItems":
		if p.Typed.Type == grammar.ValueInteger {
			t.SetMinItems(p.Typed.Integer)
		}
	case "maxItems":
		if p.Typed.Type == grammar.ValueInteger {
			t.SetMaxItems(p.Typed.Integer)
		}
	case "unique":
		if p.Typed.Type == grammar.ValueBoolean {
			t.SetUnique(p.Typed.Boolean)
		}
	case "collectionFormat":
		// Only OperationValidationBuilder knows SetCollectionFormat;
		// items.Validations does not (per survey). Type-assertion
		// guard silently drops the value for items-only targets,
		// matching v1's tagger table structure.
		//
		// Falls back to the raw value when grammar's strict
		// StringEnum rejects the input — v1 accepts any string
		// (e.g. the typo `pipe` for `pipes`) and stores verbatim.
		if ov, ok := t.(ifaces.OperationValidationBuilder); ok {
			val := p.Typed.String
			if val == "" {
				val = strings.TrimSpace(p.Value)
			}
			if val != "" {
				ov.SetCollectionFormat(val)
			}
		}
	case "enum":
		// Delegated to the existing target.SetEnum, which routes
		// through helpers.ParseEnum (post-Q1 fix: comma-list
		// trimmed, JSON array verbatim). Direct use of
		// internal/parsers/enum.Parse is deferred to the
		// post-migration cleanup commit that takes the fully-typed
		// values path; for now we preserve v1 parity.
		t.SetEnum(p.Value)
	case "default":
		t.SetDefault(p.Value)
	case "example":
		t.SetExample(p.Value)
	}
}
