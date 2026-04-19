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

func TestSpecialSchemas(t *testing.T) {
	fixturesPath := filepath.Join(scantest.FixturesDir(), "goparsing", "go123", "special")
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

	t.Run("top-level primitive declaration should render just fine", func(t *testing.T) {
		primitive, ok := sp.Definitions["primitive"]
		require.TrueT(t, ok)

		require.TrueT(t, primitive.Type.Contains("string"))
	})

	t.Run("alias to unsafe pointer at top level should render empty", func(t *testing.T) {
		uptr, ok := sp.Definitions["unsafe_pointer_alias"]
		require.TrueT(t, ok)
		var empty oaispec.Schema
		uptr.VendorExtensible = oaispec.VendorExtensible{}
		require.Equal(t, empty, uptr)
	})

	t.Run("alias to uintptr at top level should render as integer", func(t *testing.T) {
		uptr, ok := sp.Definitions["upointer_alias"]
		require.TrueT(t, ok)
		require.TrueT(t, uptr.Type.Contains("integer"))
		require.EqualT(t, "uint64", uptr.Format)
	})

	t.Run("top-level map[string]... should render just fine", func(t *testing.T) {
		gomap, ok := sp.Definitions["go_map"]
		require.TrueT(t, ok)
		require.TrueT(t, gomap.Type.Contains("object"))
		require.NotNil(t, gomap.AdditionalProperties)

		mapSchema := gomap.AdditionalProperties.Schema
		require.NotNil(t, mapSchema)
		require.TrueT(t, mapSchema.Type.Contains("integer"))
		require.EqualT(t, "uint16", mapSchema.Format)
	})

	t.Run("untagged struct referenced by a tagged model should be discovered", func(t *testing.T) {
		gostruct, ok := sp.Definitions["GoStruct"]
		require.TrueT(t, ok)
		require.TrueT(t, gostruct.Type.Contains("object"))
		require.NotEmpty(t, gostruct.Properties)

		t.Run("pointer property should render just fine", func(t *testing.T) {
			a, ok := gostruct.Properties["A"]
			require.TrueT(t, ok)
			require.TrueT(t, a.Type.Contains("number"))
			require.EqualT(t, "float", a.Format)
		})
	})

	t.Run("tagged unsupported map type should render empty", func(t *testing.T) {
		idx, ok := sp.Definitions["index_map"]
		require.TrueT(t, ok)
		var empty oaispec.Schema
		idx.VendorExtensible = oaispec.VendorExtensible{}
		require.Equal(t, empty, idx)
	})

	t.Run("redefinition of the builtin error type should render as a string", func(t *testing.T) {
		goerror, ok := sp.Definitions["go_error"]
		require.TrueT(t, ok)
		require.TrueT(t, goerror.Type.Contains("string"))

		t.Run("a type based on the error builtin should be decorated with a x-go-type: error extension", func(t *testing.T) {
			val, hasExt := goerror.Extensions.GetString("x-go-type")
			assert.TrueT(t, hasExt)
			assert.EqualT(t, "error", val)
		})
	})

	t.Run("with SpecialTypes struct", func(t *testing.T) {
		testSpecialTypesStruct(t, sp)
	})

	t.Run("with generic types", func(t *testing.T) {
		// NOTE: codescan does not really support generic types.
		// This test just makes sure generic definitions don't crash the scanner.
		//
		// The general approach of the scanner is to make an empty schema out of anything
		// it doesn't understand.

		// generic_constraint
		t.Run("generic type constraint should render like an interface", func(t *testing.T) {
			generic, ok := sp.Definitions["generic_constraint"]
			require.TrueT(t, ok)
			require.Len(t, generic.AllOf, 1) // scanner only understood one member, and skipped the ~uint16 member is doesn't understand
			member := generic.AllOf[0]
			require.TrueT(t, member.Type.Contains("object"))
			require.Len(t, member.Properties, 1)
			prop, ok := member.Properties["Uint"]
			require.TrueT(t, ok)
			require.TrueT(t, prop.Type.Contains("integer"))
			require.EqualT(t, "uint16", prop.Format)
		})

		// numerical_constraint
		t.Run("generic type constraint with union type should render an empty schema", func(t *testing.T) {
			generic, ok := sp.Definitions["numerical_constraint"]
			require.TrueT(t, ok)
			var empty oaispec.Schema
			generic.VendorExtensible = oaispec.VendorExtensible{}
			require.Equal(t, empty, generic)
		})

		// generic_map
		t.Run("generic map should render an empty schema", func(t *testing.T) {
			generic, ok := sp.Definitions["generic_map"]
			require.TrueT(t, ok)
			var empty oaispec.Schema
			generic.VendorExtensible = oaispec.VendorExtensible{}
			require.Equal(t, empty, generic)
		})

		// generic_map_alias
		t.Run("generic map alias to an anonymous generic type should render an empty schema", func(t *testing.T) {
			generic, ok := sp.Definitions["generic_map_alias"]
			require.TrueT(t, ok)
			var empty oaispec.Schema
			generic.VendorExtensible = oaispec.VendorExtensible{}
			require.Equal(t, empty, generic)
		})

		// generic_indirect
		t.Run("generic map alias to a named generic type should render a ref", func(t *testing.T) {
			generic, ok := sp.Definitions["generic_indirect"]
			require.TrueT(t, ok)
			assertIsRef(t, &generic, "#/definitions/generic_map_alias")
		})

		// generic_slice
		t.Run("generic slice should render as an array of empty schemas", func(t *testing.T) {
			generic, ok := sp.Definitions["generic_slice"]
			require.TrueT(t, ok)
			require.TrueT(t, generic.Type.Contains("array"))
			require.NotNil(t, generic.Items)
			itemsSchema := generic.Items.Schema
			require.NotNil(t, itemsSchema)
			var empty oaispec.Schema
			require.Equal(t, &empty, itemsSchema)
		})

		// union_alias:
		t.Run("alias to type constraint should render a ref", func(t *testing.T) {
			generic, ok := sp.Definitions["union_alias"]
			require.TrueT(t, ok)
			assertIsRef(t, &generic, "#/definitions/numerical_constraint")
		})
	})

	// Strip stdlib-dependent definitions whose shape drifts across Go
	// versions (e.g. reflect.Type grew iterator methods in Go 1.26). The
	// sub-tests above already pin the scanner behaviour (the fields resolve
	// to a $ref with x-go-package: reflect); the snapshot only needs to
	// cover the stable, fixture-local parts of the spec.
	for name, def := range sp.Definitions {
		if pkg, ok := def.Extensions.GetString("x-go-package"); ok && pkg == "reflect" {
			delete(sp.Definitions, name)
		}
	}

	scantest.CompareOrDumpJSON(t, sp, "go123_special_spec.json")
}

func testSpecialTypesStruct(t *testing.T, sp *oaispec.Swagger) {
	t.Helper()

	t.Run("in spite of all the pitfalls, the struct should be rendered", func(t *testing.T) {
		special, ok := sp.Definitions["special_types"]
		require.TrueT(t, ok)
		require.TrueT(t, special.Type.Contains("object"))
		props := special.Properties
		require.NotEmpty(t, props)
		require.Empty(t, special.AllOf)

		t.Run("property pointer to struct should render as a ref", func(t *testing.T) {
			ptr, ok := props["PtrStruct"]
			require.TrueT(t, ok)
			assertIsRef(t, &ptr, "#/definitions/GoStruct")
		})

		t.Run("property as time.Time should render as a formatted string", func(t *testing.T) {
			str, ok := props["ShouldBeStringTime"]
			require.TrueT(t, ok)
			require.TrueT(t, str.Type.Contains("string"))
			require.EqualT(t, "date-time", str.Format)
		})

		t.Run("property as *time.Time should also render as a formatted string", func(t *testing.T) {
			str, ok := props["ShouldAlsoBeStringTime"]
			require.TrueT(t, ok)
			require.TrueT(t, str.Type.Contains("string"))
			require.EqualT(t, "date-time", str.Format)
		})

		t.Run("property as builtin error should render as a string", func(t *testing.T) {
			goerror, ok := props["Err"]
			require.TrueT(t, ok)
			require.TrueT(t, goerror.Type.Contains("string"))

			t.Run("a type based on the error builtin should be decorated with a x-go-type: error extension", func(t *testing.T) {
				val, hasExt := goerror.Extensions.GetString("x-go-type")
				assert.TrueT(t, hasExt)
				assert.EqualT(t, "error", val)
			})
		})

		t.Run("type recognized as a text marshaler should render as a string", func(t *testing.T) {
			m, ok := props["Marshaler"]
			require.TrueT(t, ok)
			require.TrueT(t, m.Type.Contains("string"))

			t.Run("a type based on the encoding.TextMarshaler decorated with a x-go-type extension", func(t *testing.T) {
				val, hasExt := m.Extensions.GetString("x-go-type")
				assert.TrueT(t, hasExt)
				assert.EqualT(t, fixturesModule+"/goparsing/go123/special.IsATextMarshaler", val)
			})
		})

		t.Run("a json.RawMessage should be recognized and render as an object (yes this is wrong)", func(t *testing.T) {
			m, ok := props["Message"]
			require.TrueT(t, ok)
			require.TrueT(t, m.Type.Contains("object"))
		})

		t.Run("type time.Duration is not recognized as a special type and should just render as a ref", func(t *testing.T) {
			d, ok := props["Duration"]
			require.TrueT(t, ok)
			assertIsRef(t, &d, "#/definitions/Duration")

			t.Run("discovered definition should be an integer", func(t *testing.T) {
				duration, ok := sp.Definitions["Duration"]
				require.TrueT(t, ok)
				require.TrueT(t, duration.Type.Contains("integer"))
				require.EqualT(t, "int64", duration.Format)

				t.Run("time.Duration schema should be decorated with a x-go-package: time", func(t *testing.T) {
					val, hasExt := duration.Extensions.GetString("x-go-package")
					assert.TrueT(t, hasExt)
					assert.EqualT(t, "time", val)
				})
			})
		})

		testSpecialTypesStrfmt(t, props)

		t.Run("a property which is a map should render just fine, with a ref", func(t *testing.T) {
			mm, ok := props["Map"]
			require.TrueT(t, ok)
			require.TrueT(t, mm.Type.Contains("object"))
			require.NotNil(t, mm.AdditionalProperties)
			mapSchema := mm.AdditionalProperties.Schema
			require.NotNil(t, mapSchema)
			assertIsRef(t, mapSchema, "#/definitions/GoStruct")
		})

		t.Run("a property which is a named array type should render as a ref", func(t *testing.T) {
			na, ok := props["NamedArray"]
			require.TrueT(t, ok)
			assertIsRef(t, &na, "#/definitions/go_array")
		})

		testSpecialTypesWhatNot(t, sp, props)
	})
}

func testSpecialTypesStrfmt(t *testing.T, props map[string]oaispec.Schema) {
	t.Helper()

	t.Run("with strfmt types", func(t *testing.T) {
		t.Run("a strfmt.Date should be recognized and render as a formatted string", func(t *testing.T) {
			d, ok := props["FormatDate"]
			require.TrueT(t, ok)
			require.TrueT(t, d.Type.Contains("string"))
			require.EqualT(t, "date", d.Format)
		})

		t.Run("a strfmt.DateTime should be recognized and render as a formatted string", func(t *testing.T) {
			d, ok := props["FormatTime"]
			require.TrueT(t, ok)
			require.TrueT(t, d.Type.Contains("string"))
			require.EqualT(t, "date-time", d.Format)
		})

		t.Run("a strfmt.UUID should be recognized and render as a formatted string", func(t *testing.T) {
			u, ok := props["FormatUUID"]
			require.TrueT(t, ok)
			require.TrueT(t, u.Type.Contains("string"))
			require.EqualT(t, "uuid", u.Format)
		})

		t.Run("a pointer to strfmt.UUID should be recognized and render as a formatted string", func(t *testing.T) {
			u, ok := props["PtrFormatUUID"]
			require.TrueT(t, ok)
			require.TrueT(t, u.Type.Contains("string"))
			require.EqualT(t, "uuid", u.Format)
		})
	})
}

func testSpecialTypesWhatNot(t *testing.T, sp *oaispec.Swagger, props map[string]oaispec.Schema) {
	t.Helper()

	t.Run(`with the "WhatNot" anonymous inner struct`, func(t *testing.T) {
		t.Run("should render as an anonymous schema, in spite of all the unsupported things", func(t *testing.T) {
			wn, ok := props["WhatNot"]
			require.TrueT(t, ok)
			require.TrueT(t, wn.Type.Contains("object"))
			require.NotEmpty(t, wn.Properties)

			markedProps := make([]string, 0)

			for _, unsupportedProp := range []string{
				"AA", // complex128
				"A",  // complex64
				"B",  // chan int
				"C",  // func()
				"D",  // func() string
				"E",  // unsafe.Pointer
			} {
				t.Run("with property "+unsupportedProp, func(t *testing.T) {
					prop, ok := wn.Properties[unsupportedProp]
					require.TrueT(t, ok)
					markedProps = append(markedProps, unsupportedProp)

					t.Run("unsupported type in property should render as an empty schema", func(t *testing.T) {
						var empty oaispec.Schema
						require.Equal(t, empty, prop)
					})
				})
			}

			for _, supportedProp := range []string{
				"F", // uintptr
				"G",
				"H",
				"I",
				"J",
				"K",
			} {
				t.Run("with property "+supportedProp, func(t *testing.T) {
					prop, ok := wn.Properties[supportedProp]
					require.TrueT(t, ok)
					markedProps = append(markedProps, supportedProp)

					switch supportedProp {
					case "F":
						t.Run("uintptr should render as integer", func(t *testing.T) {
							require.TrueT(t, prop.Type.Contains("integer"))
							require.EqualT(t, "uint64", prop.Format)
						})
					case "G", "H":
						t.Run(
							"math/big types are not recognized as special types and as TextMarshalers they render as string",
							func(t *testing.T) {
								require.TrueT(t, prop.Type.Contains("string"))
							})
					case "I":
						t.Run("go array should render as a json array", func(t *testing.T) {
							require.TrueT(t, prop.Type.Contains("array"))
							require.NotNil(t, prop.Items)
							itemsSchema := prop.Items.Schema
							require.NotNil(t, itemsSchema)

							require.TrueT(t, itemsSchema.Type.Contains("integer"))
							// [5]byte is not recognized an array of bytes, but of uint8
							// (internally this is the same for go)
							require.EqualT(t, "uint8", itemsSchema.Format)
						})
					case "J", "K":
						t.Run("reflect types should render just fine", func(t *testing.T) {
							var dest string
							if supportedProp == "J" {
								dest = "Type"
							} else {
								dest = "Value"
							}
							assertIsRef(t, &prop, "#/definitions/"+dest)

							t.Run("the $ref should exist", func(t *testing.T) {
								deref, ok := sp.Definitions[dest]
								require.TrueT(t, ok)
								val, hasExt := deref.Extensions.GetString("x-go-package")
								assert.TrueT(t, hasExt)
								assert.EqualT(t, "reflect", val)
							})
						})
					}
				})
			}

			t.Run("we should not have any property left in WhatNot", func(t *testing.T) {
				for _, key := range markedProps {
					delete(wn.Properties, key)
				}

				require.Empty(t, wn.Properties)
			})

			t.Run("surprisingly, a tagged unexported top-level definition can be rendered", func(t *testing.T) {
				unexported, ok := sp.Definitions["unexported"]
				require.TrueT(t, ok)
				require.TrueT(t, unexported.Type.Contains("object"))
			})

			t.Run("the IsATextMarshaler type is not identified as a discovered type and is not rendered", func(t *testing.T) {
				_, ok := sp.Definitions["IsATextMarshaler"]
				require.FalseT(t, ok)
			})

			t.Run("a top-level go array should render just fine", func(t *testing.T) {
				// Notice that the semantics of fixed length are lost in this mapping
				goarray, ok := sp.Definitions["go_array"]
				require.TrueT(t, ok)
				require.TrueT(t, goarray.Type.Contains("array"))
				require.NotNil(t, goarray.Items)
				itemsSchema := goarray.Items.Schema
				require.NotNil(t, itemsSchema)
				require.TrueT(t, itemsSchema.Type.Contains("integer"))
				require.EqualT(t, "int64", itemsSchema.Format)
			})
		})
	})
}
