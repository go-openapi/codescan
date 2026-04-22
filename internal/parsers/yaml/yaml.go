// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package yaml is a thin wrapper around go.yaml.in/yaml/v3 for
// consuming the RawYAML bodies that internal/parsers/grammar/
// isolates between `---` fences.
//
// The package exists so the main grammar parser stays YAML-free and
// stdlib-only (architecture §3.3, §5.1). It is imported only by the
// analyzer layer — bridge-taggers that decide when to parse a given
// RawYAML body — never by internal/parsers/grammar/.
//
// This subpackage also establishes the sibling-sub-parser pattern:
// any future sub-language (enum-variant forms per W2, richer
// example syntax per W3, private-comment bodies per W4, …) gets its
// own `internal/parsers/<name>/` subpackage following the same seam.
package yaml

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Parse unmarshals the given raw YAML body into a generic value
// (typically a map[string]interface{} or []interface{}). The pos
// parameter is passed through any wrapping error so downstream
// diagnostics can point at the original source location — YAML
// library errors carry their own line/column numbers relative to the
// body, not to the Go source.
//
// Returns (nil, nil) for an empty body so callers can handle
// "annotation had a fence but no content" without branching on
// error-vs-nil.
//
//nolint:nilnil // (nil, nil) is the documented "empty body" return — the caller distinguishes via len(body) if needed.
func Parse(body string) (any, error) {
	if body == "" {
		return nil, nil
	}
	var v any
	if err := yaml.Unmarshal([]byte(body), &v); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	return v, nil
}

// ParseInto unmarshals body into the given destination, typically a
// pointer to a struct the caller defined to match an expected YAML
// shape (e.g., operation-body or extension-value). Wraps the
// underlying error for uniform error reporting.
func ParseInto(body string, dst any) error {
	if body == "" {
		return nil
	}
	if err := yaml.Unmarshal([]byte(body), dst); err != nil {
		return fmt.Errorf("yaml: %w", err)
	}
	return nil
}
