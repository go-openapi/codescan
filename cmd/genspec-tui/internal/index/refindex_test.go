// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// refSpecJSON references /definitions/User from three places — a property, an array's items, and a response schema
// — plus one external ref that must be recorded but not resolvable, and a ref to a path key needing RFC 6901
// escaping.
//
// Indented exactly as json.MarshalIndent renders.
const refSpecJSON = `{
  "definitions": {
    "Team": {
      "properties": {
        "lead": {
          "$ref": "#/definitions/User"
        },
        "members": {
          "items": {
            "$ref": "#/definitions/User"
          },
          "type": "array"
        },
        "logo": {
          "$ref": "https://example.com/schemas/logo.json#/Logo"
        }
      }
    },
    "User": {
      "properties": {
        "name": {
          "type": "string"
        }
      }
    }
  },
  "paths": {
    "/pets": {
      "get": {
        "responses": {
          "200": {
            "schema": {
              "$ref": "#/definitions/User"
            }
          }
        }
      }
    }
  }
}`

// 0-based rendered lines of the four $ref values above.
const (
	refLineLead     = 5
	refLineMembers  = 9
	refLineLogo     = 14
	refLineResponse = 32
	refLineUserDecl = 18
)

func TestRefIndex_FindsEveryLocalSite(t *testing.T) {
	refs := BuildJSONIndex([]byte(refSpecJSON)).Refs

	sites := refs.RefsToPointer("/definitions/User")
	require.Len(t, sites, 3, "a property, an items schema and a response schema reference User")

	// Ordered by rendered line, which is the order F3 will step through them.
	assert.Equal(t, []int{refLineLead, refLineMembers, refLineResponse},
		[]int{sites[0].Line, sites[1].Line, sites[2].Line})

	// Each site names the node HOLDING the $ref, not the $ref member itself — that is the node the user is navigating
	// to.
	assert.Equal(t, "/definitions/Team/properties/lead", sites[0].Pointer)
	assert.Equal(t, "/definitions/Team/properties/members/items", sites[1].Pointer)
	assert.Equal(t, "/paths/~1pets/get/responses/200/schema", sites[2].Pointer,
		"the path key keeps its RFC 6901 escaping")
}

func TestRefIndex_ExternalRefsAreRecordedNotResolved(t *testing.T) {
	refs := BuildJSONIndex([]byte(refSpecJSON)).Refs

	assert.Equal(t, 4, refs.Len(), "the external ref is counted")
	assert.Empty(t, refs.RefsToPointer("/Logo"),
		"an external ref must not be matched as if it were local")

	site, ok := refs.RefAt(refLineLogo)
	require.True(t, ok, "the external ref is still locatable by line")
	assert.False(t, site.Target.Local)
	assert.Empty(t, site.Target.Pointer)
	assert.Equal(t, "https://example.com/schemas/logo.json#/Logo", site.Target.Raw,
		"the raw value is preserved verbatim, so the UI can say where it points")
}

func TestRefIndex_RefAt(t *testing.T) {
	refs := BuildJSONIndex([]byte(refSpecJSON)).Refs

	site, ok := refs.RefAt(refLineLead)
	require.True(t, ok)
	assert.Equal(t, "/definitions/User", site.Target.Pointer)
	assert.True(t, site.Target.Local)

	_, ok = refs.RefAt(0) // the opening brace: no $ref there
	assert.False(t, ok)
}

// The whole point of the index is the join: a site's target must be a pointer the SpecIndex can actually resolve to a
// line.
func TestRefIndex_TargetsResolveInTheSpecIndex(t *testing.T) {
	built := BuildJSONIndex([]byte(refSpecJSON))
	spec, refs := built.Spec, built.Refs

	for _, site := range refs.RefsToPointer("/definitions/User") {
		line, ok := spec.LineForPointer(site.Target.Pointer)
		require.True(t, ok, "target %q must exist in the spec index", site.Target.Pointer)
		assert.Equal(t, refLineUserDecl, line, "/definitions/User is declared once")
	}
}

// YAML renders the same document at different lines; the pointers must match the JSON side exactly, so navigation
// survives a format toggle.
func TestRefIndex_YAMLMatchesJSONPointers(t *testing.T) {
	const refSpecYAML = `definitions:
  Team:
    properties:
      lead:
        $ref: '#/definitions/User'
      members:
        items:
          $ref: '#/definitions/User'
        type: array
  User:
    properties:
      name:
        type: string
`
	refs := BuildYAMLIndex([]byte(refSpecYAML)).Refs

	sites := refs.RefsToPointer("/definitions/User")
	require.Len(t, sites, 2)
	assert.Equal(t, "/definitions/Team/properties/lead", sites[0].Pointer)
	assert.Equal(t, "/definitions/Team/properties/members/items", sites[1].Pointer)
	assert.Equal(t, []int{4, 7}, []int{sites[0].Line, sites[1].Line},
		"same nodes, YAML line numbers")
}

func TestParseRefTarget(t *testing.T) {
	for _, c := range []struct {
		name    string
		raw     string
		local   bool
		pointer string
	}{
		{"local definition", "#/definitions/User", true, "/definitions/User"},
		{"local with escaped path key", "#/paths/~1pets/get", true, "/paths/~1pets/get"},
		{"local percent-encoded", "#/definitions/A%20B", true, "/definitions/A B"},
		{"hierarchical name", "#/definitions/pkg/Name", true, "/definitions/pkg/Name"},
		{"whole document", "#", true, ""},
		{"external file", "other.json", false, ""},
		{"external with fragment", "other.json#/definitions/User", false, ""},
		{"url", "https://example.com/s.json#/X", false, ""},
		{"plain-name fragment", "#Foo", false, ""},
		// A malformed escape must not drop the ref on the floor.
		{"bad percent escape", "#/definitions/A%zz", true, "/definitions/A%zz"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := ParseRefTarget(c.raw)
			assert.Equal(t, c.raw, got.Raw)
			assert.Equal(t, c.local, got.Local)
			assert.Equal(t, c.pointer, got.Pointer)
		})
	}
}

func TestRefIndex_NilAndEmpty(t *testing.T) {
	var nilIdx *RefIndex
	assert.Empty(t, nilIdx.RefsToPointer("/definitions/User"))
	assert.Zero(t, nilIdx.Len())
	_, ok := nilIdx.RefAt(0)
	assert.False(t, ok)

	refs := BuildJSONIndex([]byte(`{"definitions":{}}`)).Refs
	assert.Zero(t, refs.Len())
	assert.Empty(t, refs.RefsToPointer("/definitions/User"))
}
