// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The resolvers are pure over the indexes, so every case below is reachable without a Model, a panel or a temp dir
// - which is the point: this is where the "nothing here" answers are decided, and each is a different thing to tell the
// user.

const userPtr = "/definitions/User"

func TestResolveSourceToSpec(t *testing.T) {
	for _, tc := range []struct {
		name  string
		src   *index.SourceIndex
		spec  *index.SpecIndex
		file  string
		line  int
		want  string // the miss, or "" when it must resolve
		wantP string
	}{
		{
			name: "no file open",
			file: "",
			want: noFileDesc,
		},
		{
			name: "no provenance at all",
			src:  index.BuildSourceIndex(nil),
			file: "user.go", line: 1,
			want: noProvenanceDesc,
		},
		{
			name: "anchored file but line above the first anchor",
			src:  anchored("user.go", 3, userPtr),
			file: "user.go", line: 1,
			want: noAnchorDesc,
		},
		{
			// The pointer resolves on the source side but has nowhere to land: a different answer from "no source".
			name: "anchored but not rendered in this view",
			src:  anchored("user.go", 1, userPtr),
			spec: rendered("/definitions/Other"),
			file: "user.go", line: 1,
			want: userPtr + notRenderedSuffix,
		},
		{
			name: "resolves",
			src:  anchored("user.go", 1, userPtr),
			spec: rendered(userPtr),
			file: "user.go", line: 1,
			wantP: userPtr,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSourceToSpec(tc.src, tc.spec, tc.file, tc.line)

			if tc.want != "" {
				assert.False(t, got.Found)
				assert.Equal(t, tc.want, got.Miss)

				return
			}
			require.True(t, got.Found, "miss: %s", got.Miss)
			assert.Equal(t, tc.wantP, got.Pointer)
			assert.Empty(t, got.Miss, "a resolution carries no miss")
		})
	}
}

func TestResolveSpecToSource(t *testing.T) {
	t.Run("no node on this line", func(t *testing.T) {
		got := resolveSpecToSource(nil, index.BuildSourceIndex(nil), 0)

		assert.False(t, got.Found)
		assert.Equal(t, noNodeDesc, got.Miss)
	})

	t.Run("no provenance at all", func(t *testing.T) {
		got := resolveSpecToSource(rendered(userPtr), index.BuildSourceIndex(nil), 0)

		assert.False(t, got.Found)
		assert.Contains(t, got.Miss, noProvenanceDesc)
	})

	// Some other node is anchored, so the index is populated - this node simply was not produced from code, which is
	// the InputSpec overlay case.
	t.Run("spec-only node", func(t *testing.T) {
		got := resolveSpecToSource(rendered(userPtr), anchored("other.go", 1, "/definitions/Other"), 0)

		assert.False(t, got.Found)
		assert.Equal(t, userPtr+noSourceSuffix, got.Miss)
	})

	t.Run("resolves", func(t *testing.T) {
		got := resolveSpecToSource(rendered(userPtr), anchored("user.go", 7, userPtr), 0)

		require.True(t, got.Found)
		assert.Equal(t, userPtr, got.Pointer)
		assert.Equal(t, "user.go", got.Pos.Filename)
		assert.Equal(t, 7, got.Pos.Line)
	})
}

func TestResolveFileToSpec(t *testing.T) {
	t.Run("file produced nothing", func(t *testing.T) {
		got := resolveFileToSpec(index.BuildSourceIndex(nil), rendered(userPtr), "/src/user.go")

		assert.False(t, got.Found)
		assert.Equal(t, "no spec node produced by user.go", got.Miss, "the miss names the file, not its whole path")
	})

	t.Run("first node is not in this render", func(t *testing.T) {
		got := resolveFileToSpec(anchored("user.go", 1, userPtr), rendered("/definitions/Other"), "user.go")

		assert.False(t, got.Found)
		assert.Equal(t, "node not in the current spec view: "+userPtr, got.Miss)
	})

	t.Run("resolves to the file's first node", func(t *testing.T) {
		got := resolveFileToSpec(anchored("user.go", 1, userPtr), rendered(userPtr), "user.go")

		require.True(t, got.Found)
		assert.Equal(t, userPtr, got.Pointer)
		assert.Zero(t, got.Line)
	})
}

func TestResolveRefToSpec(t *testing.T) {
	built := index.BuildJSONIndex([]byte(refSpecJSON))
	refs, spec := built.Refs, built.Spec

	t.Run("no $ref on this line", func(t *testing.T) {
		got := resolveRefToSpec(refs, spec, 0) // the opening brace

		assert.False(t, got.Found)
		assert.Equal(t, "no $ref on this line", got.Miss)
	})

	// The TUI renders one document and is not a $ref resolver, so an external target is reported rather than guessed at.
	t.Run("external ref", func(t *testing.T) {
		got := resolveRefToSpec(refs, spec, refLine(t, "example.com"))

		assert.False(t, got.Found)
		assert.Contains(t, got.Miss, "external ref, not in this spec:")
	})

	t.Run("local ref whose target is absent", func(t *testing.T) {
		got := resolveRefToSpec(refs, spec, refLine(t, "Ghost"))

		assert.False(t, got.Found)
		assert.Equal(t, "/definitions/Ghost"+notRenderedSuffix, got.Miss)
	})

	t.Run("resolves", func(t *testing.T) {
		got := resolveRefToSpec(refs, spec, refLine(t, "#/definitions/User"))

		require.True(t, got.Found, "miss: %s", got.Miss)
		assert.Equal(t, userPtr, got.Pointer)

		line, ok := spec.LineForPointer(userPtr)
		require.True(t, ok)
		assert.Equal(t, line, got.Line, "it lands on the definition's own line")
	})
}

// The $ref index only exists as a product of a real render, so these go through one.
const refSpecJSON = `{
  "definitions": {
    "Team": {
      "properties": {
        "lead": {
          "$ref": "#/definitions/User"
        },
        "ghost": {
          "$ref": "#/definitions/Ghost"
        },
        "vendor": {
          "$ref": "https://example.com/schemas/x.json#/Foo"
        }
      }
    },
    "User": {
      "type": "object"
    }
  }
}`

// anchored builds a source index holding one anchor.
func anchored(file string, line int, ptr string) *index.SourceIndex {
	return index.BuildSourceIndex([]scanner.Provenance{
		{Pointer: ptr, Pos: token.Position{Filename: file, Line: line}},
	})
}

// rendered builds a spec index that knows one pointer, at line 0.
func rendered(ptr string) *index.SpecIndex {
	return index.NewSpecIndex(map[int]string{0: ptr}, map[string]int{ptr: 0})
}

// refLine is the rendered line carrying the given raw $ref target.
func refLine(t *testing.T, needle string) int {
	t.Helper()
	for i, l := range strings.Split(refSpecJSON, "\n") {
		if strings.Contains(l, needle) {
			return i
		}
	}
	t.Fatalf("no line carries %q", needle)

	return -1
}
