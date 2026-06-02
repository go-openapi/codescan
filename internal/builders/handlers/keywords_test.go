// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
)

// TestIsSimpleSchemaKeyword_AllowedSet pins the OAS v2 SimpleSchema
// allowed-keyword vocabulary in code. Any future change to the
// surface (a new SimpleSchema keyword or a removed one) must update
// this test alongside the package-level map and the README
// §simple-schema-mode entry — locking the contract down so it can't
// drift silently.
func TestIsSimpleSchemaKeyword_AllowedSet(t *testing.T) {
	want := []string{
		grammar.KwMaximum,
		grammar.KwMinimum,
		grammar.KwMultipleOf,
		grammar.KwMinLength,
		grammar.KwMaxLength,
		grammar.KwPattern,
		grammar.KwMinItems,
		grammar.KwMaxItems,
		grammar.KwUnique,
		grammar.KwCollectionFormat,
		grammar.KwDefault,
		grammar.KwExample,
		grammar.KwEnum,
		grammar.KwRequired,
	}
	for _, kw := range want {
		assert.True(t, IsSimpleSchemaKeyword(kw), "keyword %q should be SimpleSchema-legal", kw)
	}
	assert.Len(t, simpleSchemaAllowed, len(want), "simpleSchemaAllowed must match the documented surface exactly")
}

// TestIsSimpleSchemaKeyword_FullSchemaOnly pins the keywords that
// MUST be rejected as full-Schema-only. These are the keywords the
// schema Bool handler gates and emits CodeUnsupportedInSimpleSchema
// for under SimpleSchema mode.
func TestIsSimpleSchemaKeyword_FullSchemaOnly(t *testing.T) {
	forbidden := []string{
		grammar.KwReadOnly,
		grammar.KwDiscriminator,
	}
	for _, kw := range forbidden {
		assert.False(t, IsSimpleSchemaKeyword(kw), "keyword %q must NOT be SimpleSchema-legal", kw)
	}
}

// TestIsSimpleSchemaKeyword_Unknown pins the predicate's behaviour
// on an unknown keyword name — returns false, no panic.
func TestIsSimpleSchemaKeyword_Unknown(t *testing.T) {
	assert.False(t, IsSimpleSchemaKeyword("nosuchkeyword"))
	assert.False(t, IsSimpleSchemaKeyword(""))
}
