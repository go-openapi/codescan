// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

// These tests mirror the baseline coverage-enhancement tests. They scan
// dedicated fixtures under fixtures/enhancements/ and compare the result to
// the golden JSON captured on the baseline worktree, so we can catch any
// behavioural drift introduced by the refactor.

func TestCoverage_EmbeddedTypes(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/embedded-types/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_embedded_types.json")
}

func TestCoverage_AllOfEdges(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/allof-edges/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_allof_edges.json")
}

func TestCoverage_StrfmtArrays(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/strfmt-arrays/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_strfmt_arrays.json")
}

func TestCoverage_DefaultsExamples(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/defaults-examples/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_defaults_examples.json")
}

func TestCoverage_InterfaceMethods(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/interface-methods/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_interface_methods.json")
}

func TestCoverage_InterfaceMethods_XNullable(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:                []string{"./enhancements/interface-methods/..."},
		WorkDir:                 scantest.FixturesDir(),
		ScanModels:              true,
		SetXNullableForPointers: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_interface_methods_xnullable.json")
}

// TestCoverage_AliasExpand scans the alias-expand fixture with default
// Options so that buildAlias / buildFieldAlias take the non-transparent
// expansion path: each alias resolves to the underlying type and the
// target is emitted inline rather than as a $ref.
func TestCoverage_AliasExpand(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-expand/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_expand.json")
}

// TestCoverage_AliasRef scans the alias-expand fixture with RefAliases=true
// so body-parameter and response aliases resolve to $ref via makeRef, and
// the alias-of-alias chain resolves through the non-transparent switch.
func TestCoverage_AliasRef(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-expand/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_ref.json")
}

// TestCoverage_AliasResponseRef scans a fixture where the swagger:response
// annotation is itself on an alias declaration. Under RefAliases=true the
// scanner takes the responseBuilder.buildAlias refAliases switch, which
// is not covered by any other test.
func TestCoverage_AliasResponseRef(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-response/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_response_ref.json")
}

func TestCoverage_ResponseEdges(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-edges/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_edges.json")
}

func TestCoverage_NamedBasic(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/named-basic/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_named_basic.json")
}

// TestCoverage_SwaggerTypeArray exercises the fallthrough introduced by
// upstream #11: when swagger:type is set to a value not recognised by
// SwaggerSchemaForType (e.g. "array"), the builder resolves the underlying
// type instead of emitting an empty schema. Covers buildNamedSlice,
// buildNamedArray and buildNamedStruct fallthrough branches.
func TestCoverage_SwaggerTypeArray(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/swagger-type-array/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_swagger_type_array.json")
}

func TestCoverage_RefAliasChain(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/ref-alias-chain/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_ref_alias_chain.json")
}

func TestCoverage_EnumDocs(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/enum-docs/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_enum_docs.json")
}

// TestCoverage_EnumOverrides captures the v1 behavior for five
// enum-related cases that W2 needs to pin down before the P5.1
// schema-builder migration:
//
//	A. `swagger:enum` with matching consts            — const inference
//	B. inline `enum: a,b,c` only                      — inline only
//	C. inline `enum: ["a","b","c"]` JSON form only    — JSON inline only
//	D. `swagger:enum` with NO matching consts         — empty/??? case
//	E. `swagger:enum` + matching consts + inline on   — override question
//	   the field
//
// See `.claude/plans/workshops/w2-enum.md` §2.6 and
// `fixtures/enhancements/enum-overrides/types.go` for the fixture.
// The golden snapshot becomes the v1-behavior contract the v2
// migration either preserves or consciously diverges from.
func TestCoverage_EnumOverrides(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/enum-overrides/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_enum_overrides.json")
}

func TestCoverage_TextMarshal(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/text-marshal/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_text_marshal.json")
}

func TestCoverage_AllHTTPMethods(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/all-http-methods/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_all_http_methods.json")
}

// TestCoverage_UnknownAnnotation asserts that scanning a file with an
// unknown swagger: annotation returns a classifier error. This exercises
// the default branch of typeIndex.detectNodes.
func TestCoverage_UnknownAnnotation(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/unknown-annotation/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.Error(t, err)
}

func TestCoverage_NamedStructTags(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/named-struct-tags/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_named_struct_tags.json")
}

func TestCoverage_NamedStructTagsRef(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/named-struct-tags-ref/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_named_struct_tags-ref.json")
}

func TestCoverage_TopLevelKinds(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/top-level-kinds/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_top_level_kinds.json")
}

// TestCoverage_InputOverlay feeds an InputSpec carrying paths with every
// HTTP verb so that operations from the input spec are indexed before the
// scanner merges its own discoveries.
func TestCoverage_InputOverlay(t *testing.T) {
	input := &oaispec.Swagger{
		SwaggerProps: oaispec.SwaggerProps{
			Swagger: "2.0",
			Info: &oaispec.Info{
				InfoProps: oaispec.InfoProps{
					Title:   "Overlay",
					Version: "0.0.1",
				},
			},
			Paths: &oaispec.Paths{
				Paths: map[string]oaispec.PathItem{
					"/items": {
						PathItemProps: oaispec.PathItemProps{
							Get:     &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "listItems"}},
							Post:    &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "createItem"}},
							Put:     &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "replaceItem"}},
							Patch:   &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "patchItem"}},
							Delete:  &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "deleteItem"}},
							Head:    &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "checkItem"}},
							Options: &oaispec.Operation{OperationProps: oaispec.OperationProps{ID: "optionsItem"}},
						},
					},
				},
			},
		},
	}

	doc, err := codescan.Run(&codescan.Options{
		Packages:  []string{"./enhancements/embedded-types/..."},
		WorkDir:   scantest.FixturesDir(),
		InputSpec: input,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_input_overlay.json")
}
