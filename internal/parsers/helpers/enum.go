// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"encoding/json"
	"go/ast"
	"log"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
)

// ParseValueFromSchema converts a raw annotation value to the Go
// representation implied by the target schema's Type/Format. Used
// by default:/example: setters where the annotation body is a
// primitive literal whose meaning depends on the target:
// `default: 3` becomes int(3) against `Type: "integer"`, "3"
// against `Type: "string"`, and so on. JSON-typed targets
// (`object`, `array`) attempt unmarshal and fall back to the raw
// string on invalid JSON.
//
// A nil schema yields the raw string unchanged. Numeric/boolean
// parsing errors are surfaced to the caller; JSON-parse failures
// are absorbed (v1 quirk; preserved for parity).
func ParseValueFromSchema(s string, schema *spec.SimpleSchema) (any, error) {
	if schema == nil {
		return s, nil
	}

	switch strings.Trim(schema.TypeName(), "\"") {
	case "integer", "int", "int64", "int32", "int16":
		return strconv.Atoi(s)
	case "bool", "boolean":
		return strconv.ParseBool(s)
	case "number", "float64", "float32":
		return strconv.ParseFloat(s, 64)
	case "object":
		var obj map[string]any
		if err := json.Unmarshal([]byte(s), &obj); err != nil {
			return s, nil //nolint:nilerr // fallback: return raw string when JSON is invalid
		}
		return obj, nil
	case "array":
		var slice []any
		if err := json.Unmarshal([]byte(s), &slice); err != nil {
			return s, nil //nolint:nilerr // fallback: return raw string when JSON is invalid
		}
		return slice, nil
	default:
		return s, nil
	}
}

// ParseEnum turns an `enum: …` annotation value into a typed []any.
// Accepts the JSON-array form (`enum: ["a","b"]`) and the
// comma-list form (`enum: a, b`). Per-value typing is applied via
// ParseValueFromSchema against the target's scheme.
func ParseEnum(val string, s *spec.SimpleSchema) []any {
	var rawElements []json.RawMessage
	if err := json.Unmarshal([]byte(val), &rawElements); err != nil {
		log.Print("WARNING: item list for enum is not a valid JSON array, using the old deprecated format")
		return parseEnumCommaList(val, s)
	}

	out := make([]any, len(rawElements))
	for i, d := range rawElements {
		ds, err := strconv.Unquote(string(d))
		if err != nil {
			ds = string(d)
		}
		v, err := ParseValueFromSchema(ds, s)
		if err != nil {
			out[i] = ds
			continue
		}
		out[i] = v
	}
	return out
}

// parseEnumCommaList handles the legacy `enum: a, b, c` form. Per-
// value whitespace is trimmed (W2 §2.6 quirk 1 fix). Parse errors
// on individual values fall back to the raw string.
func parseEnumCommaList(val string, s *spec.SimpleSchema) []any {
	list := strings.Split(val, ",")
	out := make([]any, len(list))
	for i, d := range list {
		d = strings.TrimSpace(d)
		v, err := ParseValueFromSchema(d, s)
		if err != nil {
			out[i] = d
			continue
		}
		out[i] = v
	}
	return out
}

// GetEnumBasicLitValue converts a Go AST basic literal (the RHS of
// a `const Foo Kind = "bar"` declaration) into its runtime value —
// the representation the scanner's enum-discovery passes emit as
// enum entries on the corresponding Swagger schema.
func GetEnumBasicLitValue(basicLit *ast.BasicLit) any {
	switch basicLit.Kind.String() {
	case "INT":
		if result, err := strconv.ParseInt(basicLit.Value, 10, 64); err == nil {
			return result
		}
	case "FLOAT":
		if result, err := strconv.ParseFloat(basicLit.Value, 64); err == nil {
			return result
		}
	default:
		return strings.Trim(basicLit.Value, "\"")
	}
	return nil
}

// ExtEnumDesc is the vendor-extension key used to expose the
// per-enum-value documentation line the scanner builds from
// `swagger:enum` + const comments.
const ExtEnumDesc = "x-go-enum-desc"

// GetEnumDesc reads the x-go-enum-desc extension off a Swagger
// extensions map, if present. Empty string when absent.
func GetEnumDesc(extensions spec.Extensions) string {
	desc, _ := extensions.GetString(ExtEnumDesc)
	return desc
}

// EnumDescExtension returns the vendor-extension key. Call-sites
// use it to AddExtension / delete stale entries without hard-
// coding the string.
func EnumDescExtension() string { return ExtEnumDesc }
