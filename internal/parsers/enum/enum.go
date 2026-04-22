// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package enum is a thin sub-parser for the two surface forms of the
// `enum:` keyword — a comma-separated list and a JSON array — used
// by the v2 grammar parser's schema/parameters/responses bridge-
// taggers at P5.
//
// Like `internal/parsers/yaml/`, this subpackage follows the
// sub-parser pattern from architecture §3.3: imported only by the
// analyzer layer (bridge-taggers), never by
// `internal/parsers/grammar/`. The main grammar parser captures the
// raw enum value as `Property.Value` verbatim; the bridge-tagger
// hands that raw string to enum.Parse.
//
// See `.claude/plans/workshops/w2-enum.md` §2 for the decisions
// this package implements, and `.claude/plans/p5-builder-migrations.md`
// §6 for the API rationale.
package enum

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyOrNullArray is returned by the JSON path when the input
// parses as null or an empty array — ambiguous input the caller
// may want to surface as a warning.
var ErrEmptyOrNullArray = errors.New("JSON array is empty or null")

// Parse detects the surface form of raw and returns the parsed
// []any values. Shape detection:
//
//   - Leading `[` (after leading whitespace) → JSON-array path
//     (encoding/json). Values come out with JSON-inferred types:
//     string stays string, numbers become float64, booleans stay
//     bool, nulls become nil, objects become map[string]any,
//     arrays become []any.
//
//   - Otherwise → comma-list, each value TrimSpace'd so
//     `"red, green, blue"` produces `["red","green","blue"]`
//     (not `["red"," green"," blue"]` — this fixes v1's case-B
//     whitespace quirk per W2 §2.6).
//
// Empty/whitespace-only input returns (nil, nil): a fenced but
// empty enum is a no-op, not an error.
//
// On malformed JSON input, Parse falls back to the comma-list
// path and returns a non-nil fallbackErr describing the JSON
// issue. The returned values slice is still populated from the
// fallback. Callers typically surface the incident as a
// SeverityWarning diagnostic via the bridge-tagger rather than
// aborting the parse. This matches v1's forgiving behavior.
//
// Non-scalar enum values (objects, arrays, null) are supported
// natively through the JSON path — the returned slice carries
// whatever JSON produced. The JSONSchema / OpenAPI `enum`
// property accepts any value type; emission policy is the
// bridge-tagger's concern.
func Parse(raw string) (values []any, fallbackErr error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		values, err := parseJSON(trimmed)
		if err == nil {
			return values, nil
		}
		// JSON-looking input that failed to parse — fall back to
		// comma-list per v1 parity. Return the JSON error so the
		// caller can surface it as a warning.
		return parseCommaList(raw), fmt.Errorf("enum: %w", err)
	}
	return parseCommaList(raw), nil
}

// parseJSON unmarshals a JSON array into []any. Rejects top-level
// non-array JSON (e.g., a bare scalar, object) with an explicit
// error so the fallback path doesn't silently eat structured
// input the caller meant as an array.
func parseJSON(s string) ([]any, error) {
	var result []any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrEmptyOrNullArray
	}
	return result, nil
}

// parseCommaList splits s on `,`, trims whitespace per-value,
// and drops empty entries. The trimming is intentional and
// deliberately diverges from v1's behavior of preserving leading
// whitespace in each split segment (W2 §2.6).
func parseCommaList(s string) []any {
	parts := strings.Split(s, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
