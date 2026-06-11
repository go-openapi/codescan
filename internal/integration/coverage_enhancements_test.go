// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
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

	// Bidirectional witness — same fixture, two body-payload
	// shapes side by side:
	//
	//   - ResponseEnvelope.payload typed PayloadAlias (UNannotated)
	//     → dissolves to $ref: Payload (the unaliased target)
	//   - ResponseEnvelopeModeled.payload typed PayloadAliasModeled
	//     (annotated) → preserves $ref: PayloadAliasModeled
	//
	// Together they pin the rule: `swagger:model` is the sole
	// gate for whether an alias name surfaces in field-site $refs.
	respUnann := doc.Definitions["ResponseEnvelope"].Properties["payload"]
	assert.Equal(t, "#/definitions/Payload", respUnann.Ref.String(),
		"unannotated PayloadAlias dissolves to its unaliased target")

	require.Contains(t, doc.Definitions, "PayloadAliasModeled",
		"annotated alias must surface as a first-class definition")
	respAnn := doc.Definitions["ResponseEnvelopeModeled"].Properties["payload"]
	assert.Equal(t, "#/definitions/PayloadAliasModeled", respAnn.Ref.String(),
		"annotated PayloadAliasModeled preserves its identity in the field $ref")

	// Bidirectional response-side witness — the same pattern
	// applied to top-level swagger:response body fields. The
	// unannotated AliasedResponse and the annotated
	// AliasedModeledResponse sit on the same fixture canvas.
	assert.Equal(t, "#/definitions/ResponseEnvelope",
		doc.Responses["aliasedResponse"].Schema.Ref.String(),
		"unannotated response body alias dissolves to canonical")
	assert.Equal(t, "#/definitions/EnvelopeAliasModeled",
		doc.Responses["aliasedModeledResponse"].Schema.Ref.String(),
		"annotated response body alias preserves the alias name")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled",
		"annotated alias has its own definition")

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

// TestCoverage_WrapperDeclTypeOverride isolates Gap B' — the wrapper's
// own top-level definition does not honour `swagger:type` on the decl,
// even though the same annotation works at field reference sites.
// Pins today's broken behavior so the gap is visible until it's fixed.
func TestCoverage_WrapperDeclTypeOverride(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/wrapper-decl-type-override/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_wrapper_decl_type_override.json")

	defs := doc.Definitions

	t.Run("BareWrapperObject top-level definition is typed object", func(t *testing.T) {
		def, ok := defs["BareWrapperObject"]
		require.TrueT(t, ok)
		assert.TrueT(t, def.Type.Contains("object"), "wrapper-decl swagger:type object should produce a typed object schema")
	})

	t.Run("BareWrapperArray top-level definition is typed array", func(t *testing.T) {
		def, ok := defs["BareWrapperArray"]
		require.TrueT(t, ok)
		assert.TrueT(t, def.Type.Contains("array"), "wrapper-decl swagger:type array should produce a typed array schema")
		require.NotNil(t, def.Items)
		require.NotNil(t, def.Items.Schema)
		assert.TrueT(t, def.Items.Schema.Type.Contains("integer"), "array items reflect []byte → uint8")
	})
}

// TestCoverage_RawMessageOverride captures the user-classifier-override
// precedence for json.RawMessage. The recognizer emits an empty schema
// (`{}`, "any type") as the baseline. A user-declared wrapping type
// carrying `swagger:type` decoration overrides via the
// classifierNamedArrayLike path (RawMessage underlying is []byte).
// A field-level `swagger:type` is now consumed by scanFieldDoc and
// applied in applyFieldCarrier after buildFromType (Gap C — closed).
func TestCoverage_RawMessageOverride(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/raw-message-override/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_raw_message_override.json")

	defs := doc.Definitions

	t.Run("case A: plain json.RawMessage field emits empty schema", func(t *testing.T) {
		def, ok := defs["PlainContainer"]
		require.TrueT(t, ok)
		payload, ok := def.Properties["payload"]
		require.TrueT(t, ok)
		assert.EqualT(t, 0, len(payload.Type), "payload should have no type — empty schema")
		assert.EqualT(t, "", payload.Format)
	})

	t.Run("case B: wrapper-type swagger:type overrides at field reference sites", func(t *testing.T) {
		def, ok := defs["TypedContainer"]
		require.TrueT(t, ok)

		obj, ok := def.Properties["obj"]
		require.TrueT(t, ok)
		assert.TrueT(t, obj.Type.Contains("object"), "obj should be typed object via swagger:type on AsObject")

		arr, ok := def.Properties["arr"]
		require.TrueT(t, ok)
		assert.TrueT(t, arr.Type.Contains("array"), "arr should be typed array via swagger:type on AsArray")
		require.NotNil(t, arr.Items)
		require.NotNil(t, arr.Items.Schema)
		assert.TrueT(t, arr.Items.Schema.Type.Contains("integer"), "array items reflect []byte → uint8")
	})

	t.Run("case B': wrapper-type top-level definitions honour swagger:type", func(t *testing.T) {
		// buildFromDecl now applies classifierNamedTypeOverride from
		// s.Decl.Comments before the kind-dispatch, so the wrapper's
		// own definition reflects the decl-level swagger:type override
		// (Gap B' — closed).
		asObject, ok := defs["AsObject"]
		require.TrueT(t, ok)
		assert.TrueT(t, asObject.Type.Contains("object"), "AsObject def is typed object")

		asArray, ok := defs["AsArray"]
		require.TrueT(t, ok)
		assert.TrueT(t, asArray.Type.Contains("array"), "AsArray def is typed array")
		require.NotNil(t, asArray.Items)
		require.NotNil(t, asArray.Items.Schema)
		assert.TrueT(t, asArray.Items.Schema.Type.Contains("integer"), "AsArray items reflect underlying []byte")
	})

	t.Run("case C: field-level swagger:type overrides on json.RawMessage", func(t *testing.T) {
		// scanFieldDoc now consumes swagger:type at the field level;
		// applyFieldCarrier applies it after buildFromType so the
		// override beats the RawMessage recognizer's empty-schema default.
		def, ok := defs["FieldLevelContainer"]
		require.TrueT(t, ok)

		obj, ok := def.Properties["obj"]
		require.TrueT(t, ok)
		assert.TrueT(t, obj.Type.Contains("object"), "field-level swagger:type object overrides RawMessage default")

		arr, ok := def.Properties["arr"]
		require.TrueT(t, ok)
		assert.TrueT(t, arr.Type.Contains("array"), "field-level swagger:type array overrides RawMessage default")
		require.NotNil(t, arr.Items)
		require.NotNil(t, arr.Items.Schema)
		assert.TrueT(t, arr.Items.Schema.Type.Contains("integer"), "fallback to Underlying() yields []byte → integer/uint8 items")
	})
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

	// User annotations on alias decls. The alias-dispatch path
	// consults swagger:strfmt at the buildDeclAlias entry, so:
	//   - `type X = any` + swagger:strfmt date → `{string, date}`
	//   - `type X = int64` + swagger:strfmt uuid → `{string, uuid}`
	// The unannotated case (Wildcard) stays as the documented "any
	// value allowed" empty body, and `swagger:type` continues to
	// fire via classifierNamedTypeOverride (CountTyped baseline).
	datestamp := doc.Definitions["Datestamp"]
	assert.Equal(t, []string{"string"}, []string(datestamp.Type),
		"swagger:strfmt date on `type X = any` must produce {string, date}")
	assert.Equal(t, "date", datestamp.Format)

	userID := doc.Definitions["UserIDStrf"]
	assert.Equal(t, []string{"string"}, []string(userID.Type),
		"swagger:strfmt uuid on `type X = int64` must produce {string, uuid}")
	assert.Equal(t, "uuid", userID.Format)

	wildcard := doc.Definitions["Wildcard"]
	assert.Empty(t, wildcard.Type,
		"unannotated `type X = any` (no strfmt) keeps the open Swagger 2.0 shape")

	countTyped := doc.Definitions["CountTyped"]
	assert.Equal(t, []string{"integer"}, []string(countTyped.Type),
		"baseline: swagger:type on alias of any continues to work via classifierNamedTypeOverride")

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

// TestCoverage_GenericInstantiation exercises buildNamedType's
// generic-instantiation short-circuit. A field whose type is an
// instantiation (e.g. `GenericSlice[int]`) must emit with the
// substituted underlying shape, not as a $ref to the generic
// declaration (whose own schema is empty because type parameters
// are filtered as UnsupportedBuiltinType).
func TestCoverage_GenericInstantiation(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/generic-instantiation/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_generic_instantiation.json")
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

// TestCoverage_ParametersMapPostDecl scans a fixture that witnesses a bug
// in parameters.buildFromFieldMap: the schema sub-builder's
// PostDeclarations are not propagated to the parent parameters
// builder, so a map's value-type model registered during the build
// never reaches the spec's definitions section.
//
// The pre-fix golden shows the buggy state (LocalItem missing from
// definitions). The fix commit regenerates the golden to show LocalItem
// appearing, witnessing the resolution.
func TestCoverage_ParametersMapPostDecl(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/parameters-map-postdecl/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_parameters_map_postdecl.json")
}

// The TestCoverage_Routes* family captures the swagger:route body
// sub-language surface (`Parameters:` and `Responses:` blocks) under
// integration goldens. The legacy SetOpParams / SetOpResponses parsers
// in builders/routes are unit-tested in-package only; without these
// fixtures, retiring those parsers in favour of routebody +
// handlers.Dispatch* would lose its safety net. See M6.5-PRE plan.

func TestCoverage_RoutesParamsPath(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-path/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_path.json")
}

func TestCoverage_RoutesParamsQueryValidations(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-query-validations/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_query_validations.json")
}

func TestCoverage_RoutesParamsBodyRef(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-params-body-ref/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_body_ref.json")
}

func TestCoverage_RoutesResponsesTaggedBody(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-responses-tagged-body/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_tagged_body.json")
}

func TestCoverage_RoutesParamsQueryString(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-query-string/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_query_string.json")
}

func TestCoverage_RoutesParamsQueryNumber(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-query-number/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_query_number.json")
}

func TestCoverage_RoutesParamsQueryBoolean(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-query-boolean/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_query_boolean.json")
}

func TestCoverage_RoutesParamsQueryArray(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-query-array/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_query_array.json")
}

func TestCoverage_RoutesParamsHeaderString(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-header-string/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_header_string.json")
}

func TestCoverage_RoutesParamsFormString(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-form-string/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_form_string.json")
}

func TestCoverage_RoutesParamsBodyArray(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-params-body-array/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_body_array.json")
}

func TestCoverage_RoutesParamsBodyArrayNested(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-params-body-array-nested/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_body_array_nested.json")
}

func TestCoverage_RoutesParamsBodyWithSchemaValidations(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-params-body-with-schema-validations/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_body_with_schema_validations.json")
}

func TestCoverage_RoutesParamsMultiple(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-params-multiple/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_multiple.json")
}

func TestCoverage_RoutesParamsUnknownKey(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-unknown-key/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_unknown_key.json")
}

func TestCoverage_RoutesParamsEmptyChunk(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-params-empty-chunk/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_params_empty_chunk.json")
}

func TestCoverage_RoutesResponsesPositional(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-positional/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_positional.json")
}

func TestCoverage_RoutesResponsesTaggedResponse(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-tagged-response/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_tagged_response.json")
}

func TestCoverage_RoutesResponsesMixedBodies(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-responses-mixed-bodies/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_mixed_bodies.json")
}

func TestCoverage_RoutesResponsesDescriptionOnly(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-description-only/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_description_only.json")
}

func TestCoverage_RoutesResponsesDefault(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-default/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_default.json")
}

func TestCoverage_RoutesResponsesArray(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-responses-array/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_array.json")
}

func TestCoverage_RoutesResponsesEmptyValue(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-empty-value/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_empty_value.json")
}

func TestCoverage_RoutesResponsesDefinitionFallback(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-responses-definition-fallback/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_definition_fallback.json")
}

func TestCoverage_RoutesResponsesMultipleCodes(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-responses-multiple-codes/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_multiple_codes.json")
}

func TestCoverage_RoutesFullPetstoreShape(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-full-petstore-shape/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_full_petstore_shape.json")
}

func TestCoverage_RoutesMultiMethodSamePath(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/routes-multi-method-same-path/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_multi_method_same_path.json")
}

func TestCoverage_RoutesResponsesSpaceBodyQuirk(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-space-body-quirk/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_space_body_quirk.json")
}

func TestCoverage_RoutesResponsesRefNotFound(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-responses-ref-not-found/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_responses_ref_not_found.json")
}

func TestCoverage_RoutesDescriptionDashList(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-description-dash-list/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_description_dash_list.json")
}

func TestCoverage_RoutesDescriptionYAMLFenceAbsorb(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-description-yaml-fence-absorb/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_description_yaml_fence_absorb.json")
}

func TestCoverage_RoutesListsFlexForms(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/routes-lists-flex-forms/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_routes_lists_flex_forms.json")
}

func TestCoverage_MetaListsFlexForms(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/meta-lists-flex-forms/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	scantest.CompareOrDumpJSON(t, doc, "enhancements_meta_lists_flex_forms.json")
}
