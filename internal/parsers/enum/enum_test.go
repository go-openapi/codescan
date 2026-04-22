// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package enum_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/enum"
)

func TestParseEmpty(t *testing.T) {
	cases := []string{"", "   ", "\t\n "}
	for _, in := range cases {
		v, err := enum.Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) returned err: %v", in, err)
		}
		if v != nil {
			t.Errorf("Parse(%q): want nil, got %v", in, v)
		}
	}
}

// --- comma-list path ---

func TestParseCommaListBasic(t *testing.T) {
	v, err := enum.Parse("red,green,blue")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"red", "green", "blue"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

// TestParseCommaListTrimsWhitespace is the case-B fix from W2 §2.6:
// v1 preserves literal leading whitespace in each split segment,
// producing `["red", " green", " blue"]`. v2 diverges and trims.
func TestParseCommaListTrimsWhitespace(t *testing.T) {
	v, err := enum.Parse("red, green, blue")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"red", "green", "blue"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v (case-B whitespace fix)", v, want)
	}
}

func TestParseCommaListWithTabs(t *testing.T) {
	v, err := enum.Parse("\tred\t,\tgreen\t,\tblue\t")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"red", "green", "blue"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

func TestParseCommaListDropsEmptyEntries(t *testing.T) {
	// Trailing comma or ",," shouldn't produce empty-string values.
	v, err := enum.Parse("a, ,b,")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"a", "b"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

func TestParseCommaListSingleValue(t *testing.T) {
	v, err := enum.Parse("solo")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"solo"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

// --- JSON-array path ---

func TestParseJSONArrayStrings(t *testing.T) {
	v, err := enum.Parse(`["red","green","blue"]`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"red", "green", "blue"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

func TestParseJSONArrayNumbers(t *testing.T) {
	// JSON numbers unmarshal as float64 in Go's default json.
	v, err := enum.Parse(`[1, 2, 3.5]`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{float64(1), float64(2), 3.5}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

func TestParseJSONArrayMixedTypes(t *testing.T) {
	// Objects, arrays, null are all legal enum values per OpenAPI.
	// Survive through the JSON path.
	v, err := enum.Parse(`["s", 42, true, null, {"k":"v"}, [1,2]]`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(v) != 6 {
		t.Fatalf("want 6 elements, got %d: %v", len(v), v)
	}
	if v[0] != "s" {
		t.Errorf("v[0]: got %v want s", v[0])
	}
	if v[1] != float64(42) {
		t.Errorf("v[1]: got %v want 42", v[1])
	}
	if v[2] != true {
		t.Errorf("v[2]: got %v want true", v[2])
	}
	if v[3] != nil {
		t.Errorf("v[3]: got %v want nil", v[3])
	}
	if _, ok := v[4].(map[string]any); !ok {
		t.Errorf("v[4]: want map, got %T", v[4])
	}
	if _, ok := v[5].([]any); !ok {
		t.Errorf("v[5]: want []any, got %T", v[5])
	}
}

func TestParseJSONArrayWithCommasInStrings(t *testing.T) {
	// The JSON path handles commas inside string values correctly
	// — something the comma-list path can never do.
	v, err := enum.Parse(`["a,b","c,d","e"]`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []any{"a,b", "c,d", "e"}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("got %v want %v", v, want)
	}
}

func TestParseJSONArrayLeadingWhitespace(t *testing.T) {
	// Detection looks past leading whitespace.
	v, err := enum.Parse(`   ["a","b"]`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(v) != 2 {
		t.Errorf("want 2 values, got %d", len(v))
	}
}

// --- fallback: malformed JSON falls back to comma-list ---

func TestParseMalformedJSONFallsBack(t *testing.T) {
	// Input looks like JSON (starts with `[`) but is malformed.
	// Parse must fall back to comma-list AND return a non-nil
	// fallbackErr so the caller can surface a warning.
	v, err := enum.Parse(`[unclosed`)
	if err == nil {
		t.Fatal("want non-nil fallback err for malformed JSON")
	}
	// Fallback produced something: the raw string as-is treated as
	// a single comma-list value.
	if len(v) == 0 {
		t.Errorf("fallback values must be non-empty, got %v", v)
	}
	if !strings.HasPrefix(err.Error(), "enum:") {
		t.Errorf("error should carry the 'enum:' prefix for wrapping: %v", err)
	}
}

// TestParseDetectionIsNarrow documents the deliberate design: only
// a leading `[` triggers the JSON path. Non-bracket input is always
// comma-list, even if it happens to be valid JSON of another shape
// (object, scalar, null). This matches v1's narrow detection. A
// user writing `enum: {"k":"v"}` gets a single-value comma-list
// whose value is literally `{"k":"v"}` — they should quote-wrap
// with a JSON array (`[{"k":"v"}]`) if they want structured values.
func TestParseDetectionIsNarrow(t *testing.T) {
	cases := []struct {
		in   string
		want []any
	}{
		// JSON-object-looking input: comma-list single value.
		{`{"k":"v"}`, []any{`{"k":"v"}`}},
		// JSON null-looking input: comma-list single value "null".
		{`null`, []any{"null"}},
		// Bare number: comma-list single value "42".
		{`42`, []any{"42"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			v, err := enum.Parse(tc.in)
			if err != nil {
				t.Errorf("Parse(%q) returned err: %v (should be nil)", tc.in, err)
			}
			if !reflect.DeepEqual(v, tc.want) {
				t.Errorf("Parse(%q) = %v, want %v", tc.in, v, tc.want)
			}
		})
	}
}
