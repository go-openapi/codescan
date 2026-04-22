// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package items_test

import (
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/builders/items"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest/mocks"
)

// Each test synthesises a grammar.Block via ParseText on a raw
// comment body (no Go comment markers), constructs a recording
// MockValidationBuilder, runs ApplyBlock, and inspects the
// recorded calls.

// parseBodyToBlock returns the package's polymorphic grammar.Block
// API. Used only in tests to synthesise Block values for dispatch
// verification.
//
//nolint:ireturn // returning grammar.Block matches the package's polymorphic API by design
func parseBodyToBlock(t *testing.T, body string) grammar.Block {
	t.Helper()
	p := grammar.NewParser(token.NewFileSet())
	return p.ParseAs(grammar.AnnModel, body, token.Position{Line: 1})
}

func TestApplyBlockMaximumMinimum(t *testing.T) {
	body := "items.maximum: <10\nitems.minimum: >=0"
	b := parseBodyToBlock(t, body)

	var maxCall struct {
		val  float64
		excl bool
	}
	var minCall struct {
		val  float64
		excl bool
	}
	mock := &mocks.MockValidationBuilder{
		SetMaximumFunc: func(v float64, excl bool) { maxCall.val, maxCall.excl = v, excl },
		SetMinimumFunc: func(v float64, excl bool) { minCall.val, minCall.excl = v, excl },
	}

	items.ApplyBlock(b, mock, 1)

	if maxCall.val != 10 || !maxCall.excl {
		t.Errorf("SetMaximum: got (%v, %v), want (10, true)", maxCall.val, maxCall.excl)
	}
	if minCall.val != 0 || minCall.excl {
		t.Errorf("SetMinimum: got (%v, %v), want (0, false) — `>=` is inclusive, Op=\">=\" should map excl=false",
			minCall.val, minCall.excl)
	}
}

func TestApplyBlockIntegerKeywords(t *testing.T) {
	body := "items.minLength: 3\nitems.maxLength: 10\nitems.minItems: 1\nitems.maxItems: 100"
	b := parseBodyToBlock(t, body)

	var calls struct {
		minLen, maxLen, minItems, maxItems int64
	}
	mock := &mocks.MockValidationBuilder{
		SetMinLengthFunc: func(v int64) { calls.minLen = v },
		SetMaxLengthFunc: func(v int64) { calls.maxLen = v },
		SetMinItemsFunc:  func(v int64) { calls.minItems = v },
		SetMaxItemsFunc:  func(v int64) { calls.maxItems = v },
	}

	items.ApplyBlock(b, mock, 1)

	if calls.minLen != 3 || calls.maxLen != 10 || calls.minItems != 1 || calls.maxItems != 100 {
		t.Errorf("integer calls: got %+v", calls)
	}
}

func TestApplyBlockBooleanUnique(t *testing.T) {
	b := parseBodyToBlock(t, "items.unique: true")

	var got bool
	mock := &mocks.MockValidationBuilder{
		SetUniqueFunc: func(v bool) { got = v },
	}

	items.ApplyBlock(b, mock, 1)
	if !got {
		t.Error("SetUnique should have been called with true")
	}
}

func TestApplyBlockPattern(t *testing.T) {
	b := parseBodyToBlock(t, "items.pattern: ^[a-z]+$")

	var got string
	mock := &mocks.MockValidationBuilder{
		SetPatternFunc: func(v string) { got = v },
	}

	items.ApplyBlock(b, mock, 1)
	if got != "^[a-z]+$" {
		t.Errorf("SetPattern: got %q want %q", got, "^[a-z]+$")
	}
}

func TestApplyBlockEnum(t *testing.T) {
	b := parseBodyToBlock(t, "items.enum: red, green, blue")

	var raw string
	mock := &mocks.MockValidationBuilder{
		SetEnumFunc: func(v string) { raw = v },
	}

	items.ApplyBlock(b, mock, 1)
	// Bridge passes the raw Value; v1's ParseEnum handles splitting
	// and the Q1 whitespace-trim fix applies downstream.
	if raw != "red, green, blue" {
		t.Errorf("SetEnum: got %q", raw)
	}
}

func TestApplyBlockDefaultExample(t *testing.T) {
	b := parseBodyToBlock(t, "items.default: hello\nitems.example: world")

	var def, ex any
	mock := &mocks.MockValidationBuilder{
		SetDefaultFunc: func(v any) { def = v },
		SetExampleFunc: func(v any) { ex = v },
	}

	items.ApplyBlock(b, mock, 1)
	if def != "hello" || ex != "world" {
		t.Errorf("default/example: got %v / %v", def, ex)
	}
}

func TestApplyBlockFiltersByItemsDepth(t *testing.T) {
	// Properties at level 2 (items.items.X) should NOT fire when
	// ApplyBlock is called with level 1. This is how the schema-side
	// caller recurses into nested arrays — one ApplyBlock call per
	// depth level.
	body := "items.maximum: 5\nitems.items.maximum: 10"
	b := parseBodyToBlock(t, body)

	var calls []float64
	mock := &mocks.MockValidationBuilder{
		SetMaximumFunc: func(v float64, _ bool) { calls = append(calls, v) },
	}

	items.ApplyBlock(b, mock, 1)
	if len(calls) != 1 || calls[0] != 5 {
		t.Errorf("level 1 pass: want [5], got %v", calls)
	}

	calls = nil
	items.ApplyBlock(b, mock, 2)
	if len(calls) != 1 || calls[0] != 10 {
		t.Errorf("level 2 pass: want [10], got %v", calls)
	}

	calls = nil
	items.ApplyBlock(b, mock, 3)
	if len(calls) != 0 {
		t.Errorf("level 3 pass: want no calls, got %v", calls)
	}
}

func TestApplyBlockSkipsTypeMismatchedValues(t *testing.T) {
	// Notanumber can't parse as Number; the parser emits a
	// diagnostic upstream and leaves Typed.Type == ValueNone.
	// The bridge-tagger silently skips such properties — mirrors
	// v1's early-return-on-regex-fail behavior.
	b := parseBodyToBlock(t, "items.maximum: notanumber")

	var called bool
	mock := &mocks.MockValidationBuilder{
		SetMaximumFunc: func(v float64, excl bool) { called = true; _, _ = v, excl },
	}

	items.ApplyBlock(b, mock, 1)
	if called {
		t.Error("SetMaximum must not be called when Typed.Type is not ValueNumber")
	}
}

// Interface satisfaction compile-time check.
var _ ifaces.ValidationBuilder = (*mocks.MockValidationBuilder)(nil)
