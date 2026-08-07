// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import "testing"

// specJSON is a small rendered-spec sample exercising the cases that matter.
//
// Nested objects, an array element, and a key needing RFC 6901 escaping.
//
// Indented exactly as json.MarshalIndent renders.
const specJSON = `{
  "definitions": {
    "User": {
      "properties": {
        "email": {
          "type": "string"
        }
      },
      "required": [
        "email"
      ]
    }
  },
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets"
      }
    }
  }
}`

func TestBuildJSONIndex(t *testing.T) {
	idx := BuildJSONIndex([]byte(specJSON)).Spec

	// pointer → 0-based line (count lines in specJSON above).
	want := map[string]int{
		"/definitions":                            1,
		"/definitions/User":                       2,
		"/definitions/User/properties":            3,
		"/definitions/User/properties/email":      4,
		"/definitions/User/properties/email/type": 5,
		"/definitions/User/required":              8,
		"/definitions/User/required/0":            9,
		"/paths":                                  13,
		"/paths/~1pets":                           14, // escaped key
		"/paths/~1pets/get":                       15,
		"/paths/~1pets/get/operationId":           16,
	}
	for ptr, line := range want {
		got, ok := idx.LineForPointer(ptr)
		if !ok {
			t.Errorf("pointer %q missing from index", ptr)
			continue
		}
		if got != line {
			t.Errorf("pointer %q: line = %d, want %d", ptr, got, line)
		}
	}

	// line → pointer round-trips for a representative member line.
	if p, ok := idx.PointerAt(4); !ok || p != "/definitions/User/properties/email" {
		t.Errorf("PointerAt(4) = %q,%v; want the email property", p, ok)
	}

	// A closing-brace line (7: ` },`) carries no member; PointerAt resolves to the nearest preceding member line (the
	// email type at 5).
	if p, ok := idx.PointerAt(7); !ok || p != "/definitions/User/properties/email/type" {
		t.Errorf("PointerAt(7) nearest-preceding = %q,%v", p, ok)
	}
}

// specYAML mirrors specJSON's shape, with keys in a fixed order so line numbers are stable.
//
// The index must produce the same pointers as the JSON side.
const specYAML = `definitions:
  User:
    properties:
      email:
        type: string
    required:
      - email
paths:
  /pets:
    get:
      operationId: listPets
`

func TestBuildYAMLIndex(t *testing.T) {
	idx := BuildYAMLIndex([]byte(specYAML)).Spec

	want := map[string]int{
		"/definitions":                            0,
		"/definitions/User":                       1,
		"/definitions/User/properties":            2,
		"/definitions/User/properties/email":      3,
		"/definitions/User/properties/email/type": 4,
		"/definitions/User/required":              5,
		"/definitions/User/required/0":            6,
		"/paths":                                  7,
		"/paths/~1pets":                           8, // escaped key, same as JSON
		"/paths/~1pets/get":                       9,
		"/paths/~1pets/get/operationId":           10,
	}
	for ptr, line := range want {
		got, ok := idx.LineForPointer(ptr)
		if !ok {
			t.Errorf("pointer %q missing from YAML index", ptr)
			continue
		}
		if got != line {
			t.Errorf("pointer %q: line = %d, want %d", ptr, got, line)
		}
	}
}

func TestSpecIndexEmptyAndNil(t *testing.T) {
	var nilIdx *SpecIndex
	if _, ok := nilIdx.PointerAt(3); ok {
		t.Error("nil index PointerAt should report not-found")
	}
	if nilIdx.Len() != 0 {
		t.Error("nil index Len should be 0")
	}

	empty := BuildJSONIndex([]byte(`{}`)).Spec
	if _, ok := empty.PointerAt(0); ok {
		t.Error("empty object should index no pointers")
	}
}
