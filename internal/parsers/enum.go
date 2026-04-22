// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"encoding/json"
	"go/ast"
	"log"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
)

// ParseValueFromSchema converts a raw annotation value to the Go
// representation implied by the target schema's Type/Format. Used by
// default:/example: setters where the annotation body is a primitive
// literal whose meaning depends on the target: `default: 3` becomes
// int(3) against `Type: "integer"`, "3" against `Type: "string"`, and
// so on. JSON-typed targets (`object`, `array`) attempt unmarshal and
// fall back to the raw string on invalid JSON.
//
// A nil schema yields the raw string unchanged. Numeric/boolean
// parsing errors are surfaced to the caller; JSON-parse failures are
// absorbed (documented as a v1 quirk and currently preserved for
// parity).
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

func parseEnumOld(val string, s *spec.SimpleSchema) []any {
	// Trim per-value whitespace so `enum: low, medium, high` produces
	// ["low", "medium", "high"] rather than the v1-legacy
	// ["low", " medium", " high"] — a long-standing quirk documented
	// as W2 §2.6 quirk 1 and fixed here to converge with the v2
	// enum sub-parser's behavior before the P5 schema migration.
	// JSON-array values go through ParseEnum's quoted path (below)
	// which preserves intentional whitespace inside quotes.
	list := strings.Split(val, ",")
	interfaceSlice := make([]any, len(list))
	for i, d := range list {
		d = strings.TrimSpace(d)
		v, err := ParseValueFromSchema(d, s)
		if err != nil {
			interfaceSlice[i] = d
			continue
		}

		interfaceSlice[i] = v
	}
	return interfaceSlice
}

func ParseEnum(val string, s *spec.SimpleSchema) []any {
	// obtain the raw elements of the list to latter process them with the ParseValueFromSchema
	var rawElements []json.RawMessage
	if err := json.Unmarshal([]byte(val), &rawElements); err != nil {
		log.Print("WARNING: item list for enum is not a valid JSON array, using the old deprecated format")
		return parseEnumOld(val, s)
	}

	interfaceSlice := make([]any, len(rawElements))

	for i, d := range rawElements {
		ds, err := strconv.Unquote(string(d))
		if err != nil {
			ds = string(d)
		}

		v, err := ParseValueFromSchema(ds, s)
		if err != nil {
			interfaceSlice[i] = ds
			continue
		}

		interfaceSlice[i] = v
	}

	return interfaceSlice
}

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

const extEnumDesc = "x-go-enum-desc"

func GetEnumDesc(extensions spec.Extensions) (desc string) {
	desc, _ = extensions.GetString(extEnumDesc)
	return desc
}

func EnumDescExtension() string {
	return extEnumDesc
}
