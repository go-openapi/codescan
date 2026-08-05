// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestScanVirtualFS scans a module that exists only in memory.
//
// Nothing here is on disk and no toolchain is invoked, which is the property the WASI build and the
// browser playground both depend on.
func TestScanVirtualFS(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/api\n\ngo 1.25.0\n")},
		"models/pet.go": &fstest.MapFile{Data: []byte(`package models

// Pet describes an animal in the store.
//
// swagger:model pet
type Pet struct {
	// The pet's identifier
	// required: true
	ID int64 ` + "`json:\"id\"`" + `

	// The pet's name
	// max length: 50
	Name string ` + "`json:\"name\"`" + `
}
`)},
	}

	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./models"},
		WorkDir:    ".",
		ScanModels: true,
		FS:         tree,
		// Pinned so the expectations describe the fixture, not the platform running the test — and,
		// under a WASI guest, so the default does not silently become wasip1.
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	def, ok := doc.Definitions["pet"]
	require.True(t, ok, "expected the swagger:model to be discovered, got %v", definitionNames(doc.Definitions))

	assert.Equal(t, "Pet describes an animal in the store.", def.Title)
	assert.Equal(t, []string{"id"}, def.Required)

	name, ok := def.Properties["name"]
	require.True(t, ok)
	require.NotNil(t, name.MaxLength, "field-level validations must survive the virtual load")
	assert.Equal(t, int64(50), *name.MaxLength)
}

// TestScanVirtualFSHonoursBuildTags checks that constraint resolution reaches a virtual tree: the
// annotated model lives in a file that only builds under a tag.
func TestScanVirtualFSHonoursBuildTags(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/api\n\ngo 1.25.0\n")},
		"models/base.go": &fstest.MapFile{Data: []byte(
			"package models\n\n// swagger:model base\ntype Base struct {\n\tA string `json:\"a\"`\n}\n")},
		"models/extra.go": &fstest.MapFile{Data: []byte(
			"//go:build extras\n\npackage models\n\n// swagger:model extra\ntype Extra struct {\n\tB string `json:\"b\"`\n}\n")},
	}

	scan := func(tags string) map[string]struct{} {
		doc, err := codescan.Run(&codescan.Options{
			Packages:   []string{"./models"},
			WorkDir:    ".",
			ScanModels: true,
			FS:         tree,
			BuildTags:  tags,
			GOOS:       "linux",
			GOARCH:     "amd64",
		})
		require.NoError(t, err)

		return keySet(doc.Definitions)
	}

	withoutTag := scan("")
	assert.Contains(t, withoutTag, "base")
	assert.NotContains(t, withoutTag, "extra", "a tag-gated file must not be scanned without its tag")

	withTag := scan("extras")
	assert.Contains(t, withTag, "base")
	assert.Contains(t, withTag, "extra", "the tag must reach constraint matching inside the virtual FS")
}

func keySet[V any](m map[string]V) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}

	return out
}

func definitionNames[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}

// TestScanVirtualFSStubbedStdlib scans without any standard library at all.
//
// The tree is in memory and StubStdlib withholds GOROOT, so this scan touches no filesystem
// whatsoever — the situation a browser or a WASI guest with nothing mounted is in.
func TestScanVirtualFSStubbedStdlib(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/api\n\ngo 1.25.0\n")},
		"models/event.go": &fstest.MapFile{Data: []byte(`package models

import "time"

// Event happened at some point.
//
// swagger:model event
type Event struct {
	// When it happened
	At time.Time ` + "`json:\"at\"`" + `

	// What happened
	What string ` + "`json:\"what\"`" + `
}
`)},
	}

	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./models"},
		WorkDir:    ".",
		ScanModels: true,
		FS:         tree,
		StubStdlib: true,
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	require.NoError(t, err)

	def, ok := doc.Definitions["event"]
	require.True(t, ok, "got %v", definitionNames(doc.Definitions))

	// time.Time is recognised on identity — (package "time", type "Time") — which a synthesized type
	// still carries, so the date-time format survives having no standard library to read.
	at, ok := def.Properties["at"]
	require.True(t, ok)
	assert.Equal(t, "string", at.Type[0])
	assert.Equal(t, "date-time", at.Format)

	what, ok := def.Properties["what"]
	require.True(t, ok)
	assert.Equal(t, "string", what.Type[0])
}

// TestScanVirtualFSReportsSynthesizedImports checks that withholding the standard library is visible
// to the caller rather than silently thinning the spec.
func TestScanVirtualFSReportsSynthesizedImports(t *testing.T) {
	t.Parallel()

	tree := fstest.MapFS{
		"go.mod": &fstest.MapFile{Data: []byte("module example.com/api\n\ngo 1.25.0\n")},
		"models/event.go": &fstest.MapFile{Data: []byte(
			"package models\n\nimport (\n\t\"time\"\n\t\"example.com/gone/z\"\n)\n\n" +
				"// swagger:model event\ntype Event struct {\n\tAt time.Time `json:\"at\"`\n\tZ z.Thing `json:\"z\"`\n}\n")},
	}

	var diags []codescan.Diagnostic
	_, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./models"},
		WorkDir:      ".",
		ScanModels:   true,
		FS:           tree,
		StubStdlib:   true,
		GOOS:         "linux",
		GOARCH:       "amd64",
		OnDiagnostic: func(d codescan.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)

	bySeverity := map[string]codescan.Severity{}
	for _, d := range diags {
		if d.Code == "scan.synthesized-import" {
			for _, p := range []string{"time", "example.com/gone/z"} {
				if strings.Contains(d.Message, `"`+p+`"`) {
					bySeverity[p] = d.Severity
				}
			}
		}
	}

	// Withholding the standard library was asked for, so it is a hint. The import that is merely
	// absent is a warning: that one is usually a mounting or module-cache mistake.
	sev, ok := bySeverity["time"]
	require.True(t, ok, "no report for the withheld standard library; got %v", diags)
	assert.Equal(t, codescan.SeverityHint, sev)

	sev, ok = bySeverity["example.com/gone/z"]
	require.True(t, ok, "no report for the unresolvable import; got %v", diags)
	assert.Equal(t, codescan.SeverityWarning, sev)
}
