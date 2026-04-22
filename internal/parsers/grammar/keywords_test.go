// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import "testing"

func TestLookup(t *testing.T) {
	cases := []struct {
		input   string
		found   bool
		want    string // canonical name expected
		valType ValueType
	}{
		{"maximum", true, "maximum", ValueNumber},
		{"MAX", true, "maximum", ValueNumber},           // alias, case-insensitive
		{"max-length", true, "maxLength", ValueInteger}, // alias
		{"collection format", true, "collectionFormat", ValueStringEnum},
		{"in", true, "in", ValueStringEnum},
		{"not-a-keyword", false, "", ValueNone},
		{"", false, "", ValueNone},
		{"  maximum  ", true, "maximum", ValueNumber}, // trims whitespace
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			kw, ok := Lookup(tc.input)
			if ok != tc.found {
				t.Fatalf("Lookup(%q): found=%v want=%v", tc.input, ok, tc.found)
			}
			if !ok {
				return
			}
			if kw.Name != tc.want {
				t.Errorf("Lookup(%q): name=%q want=%q", tc.input, kw.Name, tc.want)
			}
			if kw.Value.Type != tc.valType {
				t.Errorf("Lookup(%q): valueType=%d want=%d", tc.input, kw.Value.Type, tc.valType)
			}
		})
	}
}

func TestKeywordsTableShape(t *testing.T) {
	kws := Keywords()
	if len(kws) < 30 {
		t.Fatalf("keyword table unexpectedly small: %d entries", len(kws))
	}

	// Every keyword must have a canonical name and at least one legal context.
	for _, kw := range kws {
		if kw.Name == "" {
			t.Errorf("keyword with empty Name: %+v", kw)
		}
		if len(kw.Contexts) == 0 {
			t.Errorf("keyword %q has no legal contexts", kw.Name)
		}
		if kw.Value.Type == ValueStringEnum && len(kw.Value.Values) == 0 {
			t.Errorf("keyword %q: ValueStringEnum requires Values", kw.Name)
		}
	}
}

func TestKeywordsReturnsCopy(t *testing.T) {
	a := Keywords()
	b := Keywords()
	if len(a) == 0 {
		t.Fatal("empty keyword table")
	}
	a[0].Name = "mutated"
	if b[0].Name == "mutated" {
		t.Error("Keywords() must return a defensive copy")
	}
}
