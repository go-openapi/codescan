// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package embed_basic_underlying

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The second oracle in the fixtures module, for the same reason as the first (see
// enhancements/json-tag-fidelity/wire_test.go): the types live here, so only this package can
// marshal them, and the integration test cannot import across the module boundary.
//
// It records the RAW marshalled document rather than a key set, because one subject here does not
// marshal to an object at all — MarshalHost's promoted MarshalText renders the whole struct as a
// bare string. That divergence is the point: it is the case codescan deliberately does NOT model,
// and an oracle that could only describe objects would not be able to state it.
//
// Regenerate with UPDATE_GOLDEN=1, like every other golden in the repo.
const wireGolden = "wire.golden.json"

func TestWireShapes(t *testing.T) {
	count := Count(7)

	// Non-zero values throughout, so `omitempty` never hides a key.
	subjects := map[string]any{
		"BasicHost":   BasicHost{Count: 7, Label: "l"},
		"FmtHost":     FmtHost{FmtBasic: 7, Label: "l"},
		"SliceHost":   SliceHost{Codes: Codes{"a", "b"}, Label: "l"},
		"ArrayHost":   ArrayHost{Grid: Grid{1, 2, 3, 4}, Label: "l"},
		"TaggedHost":  TaggedHost{Count: 7, Label: "l"},
		"OmittedHost": OmittedHost{Count: 7, Label: "l"},
		"PtrHost":     PtrHost{Count: &count, Label: "l"},
		"MarshalHost": MarshalHost{Token: Token{}, Label: "l"},
		"MapHost":     MapHost{Registry: Registry{"k": 1}, Label: "l"},
	}

	got := make(map[string]json.RawMessage, len(subjects))
	for name, v := range subjects {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		got[name] = raw
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
