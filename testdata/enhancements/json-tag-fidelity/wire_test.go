// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package json_tag_fidelity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// This is the only test in the fixtures module, and it earns the exception: it
// produces the ORACLE for the json-tag-fidelity corpus.
//
// The types live here, so only this package can marshal them. The integration
// test cannot import across the module boundary (the library must not gain a
// dependency on its own fixtures), so the two sides meet at a committed
// artifact instead: this test writes the wire key set, the integration test
// asserts the emitted property set matches it. Neither hard-codes an answer.
//
// Regenerate with UPDATE_GOLDEN=1, like every other golden in the repo.
const wireGolden = "wire.golden.json"

func TestWireShapes(t *testing.T) {
	// Non-zero values throughout, so `omitempty` never hides a key.
	subjects := map[string]any{
		"IgnoreShadow":      IgnoreShadow{Base: Base{ID: 1, Name: "n", Age: 42}, Age: 99},
		"RenameShadow":      RenameShadow{Base: Base{ID: 1, Name: "n", Age: 42}, Age: 99},
		"PlainIgnore":       PlainIgnore{Keep: "k", Drop: "d"},
		"DashName":          DashName{Weird: "w"},
		"DashNameOmitEmpty": DashNameOmitEmpty{Weird: "w"},
		"EmbedIgnored":      EmbedIgnored{Base: Base{ID: 1, Name: "n", Age: 42}, Extra: "x"},
	}

	got := make(map[string][]string, len(subjects))
	for name, v := range subjects {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			t.Fatalf("unmarshal %s (%s): %v", name, raw, err)
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		got[name] = keys
	}

	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	path := filepath.Join(".", wireGolden)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		const filePerm = 0o600
		if err := os.WriteFile(path, data, filePerm); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", wireGolden)

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing %s — run with UPDATE_GOLDEN=1 to create: %v", wireGolden, err)
	}
	if !reflect.DeepEqual(string(want), string(data)) {
		t.Errorf("wire shapes drifted.\nwant:\n%s\ngot:\n%s", want, data)
	}
}
