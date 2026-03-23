// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func TestAliasedSchemas(t *testing.T) {
	fixturesPath := filepath.Join(scantest.FixturesDir(), "goparsing", "go123", "aliased", "schema")
	var sp *oaispec.Swagger
	t.Run("end-to-end source scan should succeed", func(t *testing.T) {
		var err error
		sp, err = codescan.Run(&codescan.Options{
			WorkDir:    fixturesPath,
			BuildTags:  "testscanner", // fixture code is excluded from normal build
			ScanModels: true,
			RefAliases: true,
		})
		require.NoError(t, err)
	})

	if enableSpecOutput {
		// for debugging, output the resulting spec as YAML
		yml, err := marshalToYAMLFormat(sp)
		require.NoError(t, err)

		_, _ = os.Stdout.Write(yml)
	}

	t.Run("type aliased to any should yield an empty schema", func(t *testing.T) {
		anything, ok := sp.Definitions["Anything"]
		require.TrueT(t, ok)

		assertHasGoPackageExt(t, anything)
		assertHasTitle(t, anything)

		// after stripping extension and title, should be empty
		anything.VendorExtensible = oaispec.VendorExtensible{}
		anything.Title = ""
		assert.Equal(t, oaispec.Schema{}, anything)
	})

	t.Run("type aliased to an empty struct should yield an empty object", func(t *testing.T) {
		empty, ok := sp.Definitions["Empty"]
		require.TrueT(t, ok)

		assertHasGoPackageExt(t, empty)
		assertHasTitle(t, empty)

		// after stripping extension and title, should be empty
		empty.VendorExtensible = oaispec.VendorExtensible{}
		empty.Title = ""
		emptyObject := &oaispec.Schema{}
		emptyObject = emptyObject.Typed("object", "").WithProperties(map[string]oaispec.Schema{})
		assert.Equal(t, *emptyObject, empty)
	})

	t.Run("struct fields defined as any or interface{} should yield properties with an empty schema", func(t *testing.T) {
		testAliasedExtendedIDAllOf(t, sp)
	})

	t.Run("aliased primitive types remain unaffected", func(t *testing.T) {
		uuid, ok := sp.Definitions["UUID"]
		require.TrueT(t, ok)

		assertHasGoPackageExt(t, uuid)
		assertHasTitle(t, uuid)

		// after strip extension, should be equal to integer with format
		uuid.VendorExtensible = oaispec.VendorExtensible{}
		uuid.Title = ""
		intSchema := &oaispec.Schema{}
		intSchema = intSchema.Typed("integer", "int64")
		assert.Equal(t, *intSchema, uuid)
	})

	t.Run("with struct having fields aliased to any or interface{}", func(t *testing.T) {
		order, ok := sp.Definitions["order"]
		require.TrueT(t, ok)

		t.Run("field defined on an alias should produce a ref", func(t *testing.T) {
			t.Run("with alias to any", func(t *testing.T) {
				_, ok = order.Properties["DeliveryOption"]
				require.TrueT(t, ok)
				scantest.AssertRef(t, &order, "DeliveryOption", "", "#/definitions/Anything") // points to an alias to any
			})

			t.Run("with alias to primitive type", func(t *testing.T) {
				_, ok = order.Properties["id"]
				require.TrueT(t, ok)
				scantest.AssertRef(t, &order, "id", "", "#/definitions/UUID") // points to an alias to any
			})

			t.Run("with alias to struct type", func(t *testing.T) {
				_, ok = order.Properties["extended_id"]
				require.TrueT(t, ok)
				scantest.AssertRef(t, &order, "extended_id", "", "#/definitions/ExtendedID") // points to an alias to any
			})

			t.Run("inside anonymous array", func(t *testing.T) {
				items, ok := order.Properties["items"]
				require.TrueT(t, ok)

				require.NotNil(t, items)
				require.NotNil(t, items.Items)

				assert.TrueT(t, items.Type.Contains("array"))
				t.Run("field as any should render as empty object", func(t *testing.T) {
					require.NotNil(t, items.Items.Schema)
					itemsSchema := items.Items.Schema
					assert.TrueT(t, itemsSchema.Type.Contains("object"))

					require.MapContainsT(t, itemsSchema.Properties, "extra_options")
					extraOptions := itemsSchema.Properties["extra_options"]
					assertHasExtension(t, extraOptions, "x-go-name")

					extraOptions.VendorExtensible = oaispec.VendorExtensible{}
					empty := oaispec.Schema{}
					assert.Equal(t, empty, extraOptions)
				})
			})
		})

		t.Run("struct field defined as any should produce an empty schema", func(t *testing.T) {
			extras, ok := order.Properties["Extras"]
			require.TrueT(t, ok)
			assert.Equal(t, oaispec.Schema{}, extras)
		})

		t.Run("struct field defined as interface{} should produce an empty schema", func(t *testing.T) {
			extras, ok := order.Properties["MoreExtras"]
			require.TrueT(t, ok)
			assert.Equal(t, oaispec.Schema{}, extras)
		})
	})

	t.Run("type redefinitions and syntactic aliases to any should render the same", func(t *testing.T) {
		whatnot, ok := sp.Definitions["whatnot"]
		require.TrueT(t, ok)
		// after strip extension, should be empty
		whatnot.VendorExtensible = oaispec.VendorExtensible{}
		assert.Equal(t, oaispec.Schema{}, whatnot)

		whatnotAlias, ok := sp.Definitions["whatnot_alias"]
		require.TrueT(t, ok)
		// after strip extension, should be empty
		whatnotAlias.VendorExtensible = oaispec.VendorExtensible{}
		assert.Equal(t, oaispec.Schema{}, whatnotAlias)

		whatnot2, ok := sp.Definitions["whatnot2"]
		require.TrueT(t, ok)
		// after strip extension, should be empty
		whatnot2.VendorExtensible = oaispec.VendorExtensible{}
		assert.Equal(t, oaispec.Schema{}, whatnot2)

		whatnot2Alias, ok := sp.Definitions["whatnot2_alias"]
		require.TrueT(t, ok)
		// after strip extension, should be empty
		whatnot2Alias.VendorExtensible = oaispec.VendorExtensible{}
		assert.Equal(t, oaispec.Schema{}, whatnot2Alias)
	})

	t.Run("alias to another alias is resolved as a ref", func(t *testing.T) {
		void, ok := sp.Definitions["void"]
		require.TrueT(t, ok)

		assertIsRef(t, &void, "#/definitions/Empty") // points to another alias
	})

	t.Run("type redefinition to anonymous is not an alias and is resolved as an object", func(t *testing.T) {
		empty, ok := sp.Definitions["empty_redefinition"]
		require.TrueT(t, ok)

		assertHasGoPackageExt(t, empty)
		assertHasNoTitle(t, empty)

		// after stripping extension and title, should be empty
		empty.VendorExtensible = oaispec.VendorExtensible{}
		emptyObject := &oaispec.Schema{}
		emptyObject = emptyObject.Typed("object", "").WithProperties(map[string]oaispec.Schema{})
		assert.Equal(t, *emptyObject, empty)
	})

	t.Run("alias to a named interface should render as a $ref", func(t *testing.T) {
		iface, ok := sp.Definitions["iface_alias"]
		require.TrueT(t, ok)

		assertIsRef(t, &iface, "#/definitions/iface") // points to an interface
	})

	t.Run("interface redefinition is not an alias and should render as a $ref", func(t *testing.T) {
		iface, ok := sp.Definitions["iface_redefinition"]
		require.TrueT(t, ok)

		assertIsRef(t, &iface, "#/definitions/iface") // points to an interface
	})

	t.Run("anonymous interface should render a schema", func(t *testing.T) {
		iface, ok := sp.Definitions["anonymous_iface"]
		require.TrueT(t, ok)

		require.NotEmpty(t, iface.Properties)
		require.MapContainsT(t, iface.Properties, "String")
	})

	t.Run("anonymous struct should render as an anonymous schema", func(t *testing.T) {
		obj, ok := sp.Definitions["anonymous_struct"]
		require.TrueT(t, ok)

		require.NotEmpty(t, obj.Properties)
		require.MapContainsT(t, obj.Properties, "A")

		a := obj.Properties["A"]
		assert.TrueT(t, a.Type.Contains("object"))
		require.MapContainsT(t, a.Properties, "B")
		b := a.Properties["B"]
		assert.TrueT(t, b.Type.Contains("integer"))
	})

	t.Run("standalone model with a tag should be rendered", func(t *testing.T) {
		shouldSee, ok := sp.Definitions["ShouldSee"]
		require.TrueT(t, ok)
		assert.TrueT(t, shouldSee.Type.Contains("boolean"))
	})

	t.Run("standalone model without a tag should not be rendered", func(t *testing.T) {
		_, ok := sp.Definitions["ShouldNotSee"]
		require.FalseT(t, ok)

		_, ok = sp.Definitions["ShouldNotSeeSlice"]
		require.FalseT(t, ok)

		_, ok = sp.Definitions["ShouldNotSeeMap"]
		require.FalseT(t, ok)
	})

	t.Run("with aliases in slices and arrays", func(t *testing.T) {
		t.Run("slice redefinition should render as schema", func(t *testing.T) {
			t.Run("with anonymous slice", func(t *testing.T) {
				slice, ok := sp.Definitions["slice_type"] // []any
				require.TrueT(t, ok)
				assert.TrueT(t, slice.Type.Contains("array"))
				require.NotNil(t, slice.Items)
				require.NotNil(t, slice.Items.Schema)

				assert.Equal(t, &oaispec.Schema{}, slice.Items.Schema)
			})

			t.Run("with anonymous struct", func(t *testing.T) {
				slice, ok := sp.Definitions["slice_of_structs"] // type X = []struct{}
				require.TrueT(t, ok)
				assert.TrueT(t, slice.Type.Contains("array"))

				require.NotNil(t, slice.Items)
				require.NotNil(t, slice.Items.Schema)

				emptyObject := &oaispec.Schema{}
				emptyObject = emptyObject.Typed("object", "").WithProperties(map[string]oaispec.Schema{})
				assert.Equal(t, emptyObject, slice.Items.Schema)
			})
		})

		t.Run("alias to anonymous slice should render as schema", func(t *testing.T) {
			t.Run("with anonymous slice", func(t *testing.T) {
				slice, ok := sp.Definitions["slice_alias"] // type X = []any
				require.TrueT(t, ok)
				assert.TrueT(t, slice.Type.Contains("array"))

				require.NotNil(t, slice.Items)
				require.NotNil(t, slice.Items.Schema)

				assert.Equal(t, &oaispec.Schema{}, slice.Items.Schema)
			})

			t.Run("with anonymous struct", func(t *testing.T) {
				slice, ok := sp.Definitions["slice_of_structs_alias"] // type X = []struct{}
				require.TrueT(t, ok)
				assert.TrueT(t, slice.Type.Contains("array"))
				require.NotNil(t, slice.Items)
				require.NotNil(t, slice.Items.Schema)

				emptyObject := &oaispec.Schema{}
				emptyObject = emptyObject.Typed("object", "").WithProperties(map[string]oaispec.Schema{})
				assert.Equal(t, emptyObject, slice.Items.Schema)
			})
		})

		t.Run("alias to named alias to anonymous slice should render as ref", func(t *testing.T) {
			slice, ok := sp.Definitions["slice_to_slice"] // type X = Slice
			require.TrueT(t, ok)
			assertIsRef(t, &slice, "#/definitions/slice_type") // points to a named alias
		})
	})

	t.Run("with aliases in interfaces", func(t *testing.T) {
		testAliasedInterfaceVariants(t, sp)
	})

	t.Run("with aliases in embedded types", func(t *testing.T) {
		testAliasedEmbeddedTypes(t, sp)
	})

	scantest.CompareOrDumpJSON(t, sp, "go123_aliased_spec.json")
}

func testAliasedExtendedIDAllOf(t *testing.T, sp *oaispec.Swagger) {
	t.Helper()
	extended, ok := sp.Definitions["ExtendedID"]
	require.TrueT(t, ok)

	t.Run("struct with an embedded alias should render as allOf", func(t *testing.T) {
		require.Len(t, extended.AllOf, 2)
		assertHasTitle(t, extended)

		foundAliased := false
		foundProps := false
		for idx, member := range extended.AllOf {
			isProps := len(member.Properties) > 0
			isAlias := member.Ref.String() != ""

			switch {
			case isProps:
				props := member
				t.Run("with property of type any", func(t *testing.T) {
					evenMore, ok := props.Properties["EvenMore"]
					require.TrueT(t, ok)
					assert.Equal(t, oaispec.Schema{}, evenMore)
				})

				t.Run("with property of type interface{}", func(t *testing.T) {
					evenMore, ok := props.Properties["StillMore"]
					require.TrueT(t, ok)
					assert.Equal(t, oaispec.Schema{}, evenMore)
				})

				t.Run("non-aliased properties remain unaffected", func(t *testing.T) {
					more, ok := props.Properties["more"]
					require.TrueT(t, ok)

					assertHasExtension(t, more, "x-go-name") // because we have a struct tag
					assertHasNoTitle(t, more)

					// after stripping extension and title, should be empty
					more.VendorExtensible = oaispec.VendorExtensible{}

					strSchema := &oaispec.Schema{}
					strSchema = strSchema.Typed("string", "")
					assert.Equal(t, *strSchema, more)
				})
				foundProps = true
			case isAlias:
				assertIsRef(t, &member, "#/definitions/Empty")
				foundAliased = true
			default:
				assert.Failf(t, "embedded members in struct are not as expected", "unexpected member in allOf: %d", idx)
			}
		}
		require.TrueT(t, foundProps)
		require.TrueT(t, foundAliased)
	})
}

func testAliasedInterfaceVariants(t *testing.T, sp *oaispec.Swagger) {
	t.Helper()

	t.Run("should render anonymous interface as a schema", func(t *testing.T) {
		iface, ok := sp.Definitions["anonymous_iface"] // e.g. type X interface{ String() string}
		require.TrueT(t, ok)

		require.TrueT(t, iface.Type.Contains("object"))
		require.MapContainsT(t, iface.Properties, "String")
		prop := iface.Properties["String"]
		require.TrueT(t, prop.Type.Contains("string"))
		assert.Len(t, iface.Properties, 1)
	})

	t.Run("alias to an anonymous interface should render as a $ref", func(t *testing.T) {
		iface, ok := sp.Definitions["anonymous_iface_alias"]
		require.TrueT(t, ok)

		assertIsRef(t, &iface, "#/definitions/anonymous_iface") // points to an anonymous interface
	})

	t.Run("named interface should render as a schema", func(t *testing.T) {
		iface, ok := sp.Definitions["iface"]
		require.TrueT(t, ok)

		require.TrueT(t, iface.Type.Contains("object"))
		require.MapContainsT(t, iface.Properties, "Get")
		prop := iface.Properties["Get"]
		require.TrueT(t, prop.Type.Contains("string"))
		assert.Len(t, iface.Properties, 1)
	})

	t.Run("named interface with embedded types should render as allOf", func(t *testing.T) {
		iface, ok := sp.Definitions["iface_embedded"]
		require.TrueT(t, ok)

		require.Len(t, iface.AllOf, 2)
		foundEmbedded := false
		foundMethod := false
		for idx, member := range iface.AllOf {
			require.TrueT(t, member.Type.Contains("object"))
			require.NotEmpty(t, member.Properties)
			require.Len(t, member.Properties, 1)
			propGet, isEmbedded := member.Properties["Get"]
			propMethod, isMethod := member.Properties["Dump"]

			switch {
			case isEmbedded:
				assert.TrueT(t, propGet.Type.Contains("string"))
				foundEmbedded = true
			case isMethod:
				assert.TrueT(t, propMethod.Type.Contains("array"))
				foundMethod = true
			default:
				assert.Failf(t, "embedded members in interface are not as expected", "unexpected member in allOf: %d", idx)
			}
		}
		require.TrueT(t, foundEmbedded)
		require.TrueT(t, foundMethod)
	})

	t.Run("named interface with embedded anonymous interface should render as allOf", func(t *testing.T) {
		iface, ok := sp.Definitions["iface_embedded_anonymous"]
		require.TrueT(t, ok)

		require.Len(t, iface.AllOf, 2)
		foundEmbedded := false
		foundAnonymous := false
		for idx, member := range iface.AllOf {
			require.TrueT(t, member.Type.Contains("object"))
			require.NotEmpty(t, member.Properties)
			require.Len(t, member.Properties, 1)
			propGet, isEmbedded := member.Properties["String"]
			propAnonymous, isAnonymous := member.Properties["Error"]

			switch {
			case isEmbedded:
				assert.TrueT(t, propGet.Type.Contains("string"))
				foundEmbedded = true
			case isAnonymous:
				assert.TrueT(t, propAnonymous.Type.Contains("string"))
				foundAnonymous = true
			default:
				assert.Failf(t, "embedded members in interface are not as expected", "unexpected member in allOf: %d", idx)
			}
		}
		require.TrueT(t, foundEmbedded)
		require.TrueT(t, foundAnonymous)
	})

	t.Run("composition of empty interfaces is rendered as an empty schema", func(t *testing.T) {
		iface, ok := sp.Definitions["iface_embedded_empty"]
		require.TrueT(t, ok)

		iface.VendorExtensible = oaispec.VendorExtensible{}
		assert.Equal(t, oaispec.Schema{}, iface)
	})

	t.Run("interface embedded with an alias should be rendered as allOf, with a ref", func(t *testing.T) {
		iface, ok := sp.Definitions["iface_embedded_with_alias"]
		require.TrueT(t, ok)

		require.Len(t, iface.AllOf, 3)
		foundEmbedded := false
		foundEmbeddedAnon := false
		foundRef := false
		for idx, member := range iface.AllOf {
			propGet, isEmbedded := member.Properties["String"]
			propAnonymous, isAnonymous := member.Properties["Dump"]
			isRef := member.Ref.String() != ""

			switch {
			case isEmbedded:
				require.TrueT(t, member.Type.Contains("object"))
				require.Len(t, member.Properties, 1)
				assert.TrueT(t, propGet.Type.Contains("string"))
				foundEmbedded = true
			case isAnonymous:
				require.TrueT(t, member.Type.Contains("object"))
				require.Len(t, member.Properties, 1)
				assert.TrueT(t, propAnonymous.Type.Contains("array"))
				foundEmbeddedAnon = true
			case isRef:
				require.Empty(t, member.Properties)
				assertIsRef(t, &member, "#/definitions/iface_alias")
				foundRef = true
			default:
				assert.Failf(t, "embedded members in interface are not as expected", "unexpected member in allOf: %d", idx)
			}
		}
		require.TrueT(t, foundEmbedded)
		require.TrueT(t, foundEmbeddedAnon)
		require.TrueT(t, foundRef)
	})
}

func testAliasedEmbeddedTypes(t *testing.T, sp *oaispec.Swagger) {
	t.Helper()

	t.Run("embedded alias should render as a $ref", func(t *testing.T) {
		iface, ok := sp.Definitions["embedded_with_alias"]
		require.TrueT(t, ok)

		require.Len(t, iface.AllOf, 3)
		foundAnything := false
		foundUUID := false
		foundProps := false
		for idx, member := range iface.AllOf {
			isProps := len(member.Properties) > 0
			isRef := member.Ref.String() != ""

			switch {
			case isProps:
				require.TrueT(t, member.Type.Contains("object"))
				require.Len(t, member.Properties, 3)
				assert.MapContainsT(t, member.Properties, "EvenMore")
				foundProps = true
			case isRef:
				switch member.Ref.String() {
				case "#/definitions/Anything":
					foundAnything = true
				case "#/definitions/UUID":
					foundUUID = true
				default:
					assert.Failf(t,
						"embedded members in interface are not as expected", "unexpected $ref for member (%v): %d",
						member.Ref, idx,
					)
				}
			default:
				assert.Failf(t, "embedded members in interface are not as expected", "unexpected member in allOf: %d", idx)
			}
		}
		require.TrueT(t, foundAnything)
		require.TrueT(t, foundUUID)
		require.TrueT(t, foundProps)
	})
}
