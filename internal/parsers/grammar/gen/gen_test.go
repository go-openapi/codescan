// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// TestGeneratedDocIsCurrent verifies that re-running the generator
// against the current keyword table produces a byte-identical
// docs/annotation-keywords.md. If this test fails, run:
//
//	go generate ./internal/parsers/grammar/...
//
// and commit the updated docs file.
func TestGeneratedDocIsCurrent(t *testing.T) {
	// Test runs in internal/parsers/grammar/gen/, so climb 4 levels.
	docPath := filepath.Join("..", "..", "..", "..", "docs", "annotation-keywords.md")

	committed, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read committed doc: %v", err)
	}

	var got bytes.Buffer
	Render(&got, grammar.Keywords())

	if !bytes.Equal(committed, got.Bytes()) {
		t.Fatalf(
			"docs/annotation-keywords.md is out of sync with keywords_table.go.\n"+
				"Regenerate with: go generate ./internal/parsers/grammar/...\n"+
				"(committed=%d bytes, generated=%d bytes)",
			len(committed), got.Len(),
		)
	}
}
