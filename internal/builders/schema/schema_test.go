// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

const (
	epsilon = 1e-9

	// fixturesModule is the module path of the fixtures nested module.
	fixturesModule = "github.com/go-openapi/codescan/fixtures"
	// classificationOrderRef is the fully-qualified $ref of the classification
	// `order` model — emitted by the schema builder before the spec reduce
	// stage shortens it (see scanner.EntityDecl.DefKey).
	classificationOrderRef = "#/definitions/" + fixturesModule + "/goparsing/classification/models/order"
	fixtureMinimal3125     = "bugs/3125/minimal"
	sampleValue1           = "value1"
	sampleValue2           = "value2"
)

// NOTE: the per-type petstore schema snapshots (Tag / Pet / Order) moved to
// the full-pipeline integration golden petstore_spec.json, which captures the
// same models in their reduced form. Builder-unit tests assert properties; they
// no longer dump whole-spec goldens. See .claude/plans/golden-unit-to-integration.md.

func TestBuilder(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "NoModel")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "NoModel")]

	assert.Equal(t, oaispec.StringOrArray([]string{"object"}), schema.Type)
	assert.EqualT(t, "NoModel is a struct without an annotation.", schema.Title)
	assert.EqualT(t,
		"NoModel exists in a package\nbut is not annotated with the swagger model annotations\nso it should now show up in a test.",
		schema.Description,
	)
	assert.Len(t, schema.Required, 3)
	assert.Len(t, schema.Properties, 12)

	scantest.AssertProperty(t, &schema, "integer", "id", "int64", "ID")
	prop, ok := schema.Properties["id"]
	assert.EqualT(t,
		"ID of this no model instance.\nids in this application start at 11 and are smaller than 1000",
		prop.Description,
	)
	assert.TrueT(t, ok, "should have had an 'id' property")
	assert.InDeltaT(t, 1000.00, *prop.Maximum, epsilon)
	assert.TrueT(t, prop.ExclusiveMaximum, "'id' should have had an exclusive maximum")
	assert.NotNil(t, prop.Minimum)
	assert.InDeltaT(t, 10.00, *prop.Minimum, epsilon)
	assert.TrueT(t, prop.ExclusiveMinimum, "'id' should have had an exclusive minimum")
	assert.Equal(t, 11, prop.Default, "ID default value is incorrect")

	scantest.AssertProperty(t, &schema, "string", "NoNameOmitEmpty", "", "")
	prop, ok = schema.Properties["NoNameOmitEmpty"]
	assert.EqualT(t, "A field which has omitempty set but no name", prop.Description)
	assert.TrueT(t, ok, "should have had an 'NoNameOmitEmpty' property")

	scantest.AssertProperty(t, &schema, "string", "noteb64", "byte", "Note")
	prop, ok = schema.Properties["noteb64"]
	assert.TrueT(t, ok, "should have a 'noteb64' property")
	assert.Nil(t, prop.Items)

	scantest.AssertProperty(t, &schema, "integer", "score", "int32", "Score")
	prop, ok = schema.Properties["score"]
	assert.EqualT(t, "The Score of this model", prop.Description)
	assert.TrueT(t, ok, "should have had a 'score' property")
	assert.InDeltaT(t, 45.00, *prop.Maximum, epsilon)
	assert.FalseT(t, prop.ExclusiveMaximum, "'score' should not have had an exclusive maximum")
	assert.NotNil(t, prop.Minimum)
	assert.InDeltaT(t, 3.00, *prop.Minimum, epsilon)
	assert.FalseT(t, prop.ExclusiveMinimum, "'score' should not have had an exclusive minimum")
	assert.EqualValues(t, 27, prop.Example)
	require.NotNil(t, prop.MultipleOf, "'score' should have had a multipleOf")
	assert.InDeltaT(t, 3.00, *prop.MultipleOf, epsilon, "'score' should have had multipleOf 3")

	expectedNameExtensions := oaispec.Extensions{
		"x-go-name": "Name",
		"x-property-array": []any{
			sampleValue1,
			sampleValue2,
		},
		"x-property-array-obj": []any{
			map[string]any{
				"name":  "obj",
				"value": "field",
			},
		},
		"x-property-value": "value",
	}

	scantest.AssertProperty(t, &schema, "string", "name", "", "Name")
	prop, ok = schema.Properties["name"]
	assert.TrueT(t, ok)
	assert.EqualT(t, "Name of this no model instance", prop.Description)
	require.NotNil(t, prop.MinLength)
	require.NotNil(t, prop.MaxLength)
	assert.EqualT(t, int64(4), *prop.MinLength)
	assert.EqualT(t, int64(50), *prop.MaxLength)
	assert.EqualT(t, "[A-Za-z0-9-.]*", prop.Pattern)
	assert.Equal(t, expectedNameExtensions, prop.Extensions)

	scantest.AssertProperty(t, &schema, "string", "created", "date-time", "Created")
	prop, ok = schema.Properties["created"]
	assert.EqualT(t, "Created holds the time when this entry was created", prop.Description)
	assert.TrueT(t, ok, "should have a 'created' property")
	assert.TrueT(t, prop.ReadOnly, "'created' should be read only")

	scantest.AssertProperty(t, &schema, "string", "gocreated", "date-time", "GoTimeCreated")
	prop, ok = schema.Properties["gocreated"]
	assert.EqualT(t, "GoTimeCreated holds the time when this entry was created in go time.Time", prop.Description)
	assert.TrueT(t, ok, "should have a 'gocreated' property")

	scantest.AssertArrayProperty(t, &schema, "string", "foo_slice", "", "FooSlice")
	prop, ok = schema.Properties["foo_slice"]
	assert.EqualT(t, "a FooSlice has foos which are strings", prop.Description)
	assert.TrueT(t, ok, "should have a 'foo_slice' property")
	require.NotNil(t, prop.Items, "foo_slice should have had an items property")
	require.NotNil(t, prop.Items.Schema, "foo_slice.items should have had a schema property")
	assert.TrueT(t, prop.UniqueItems, "'foo_slice' should have unique items")
	assert.EqualT(t, int64(3), *prop.MinItems, "'foo_slice' should have had 3 min items")
	assert.EqualT(t, int64(10), *prop.MaxItems, "'foo_slice' should have had 10 max items")
	itprop := prop.Items.Schema
	assert.EqualT(t, int64(3), *itprop.MinLength, "'foo_slice.items.minLength' should have been 3")
	assert.EqualT(t, int64(10), *itprop.MaxLength, "'foo_slice.items.maxLength' should have been 10")
	assert.EqualT(t, "\\w+", itprop.Pattern, "'foo_slice.items.pattern' should have \\w+")

	scantest.AssertArrayProperty(t, &schema, "string", "time_slice", "date-time", "TimeSlice")
	prop, ok = schema.Properties["time_slice"]
	assert.EqualT(t, "a TimeSlice is a slice of times", prop.Description)
	assert.TrueT(t, ok, "should have a 'time_slice' property")
	require.NotNil(t, prop.Items, "time_slice should have had an items property")
	require.NotNil(t, prop.Items.Schema, "time_slice.items should have had a schema property")
	assert.TrueT(t, prop.UniqueItems, "'time_slice' should have unique items")
	assert.EqualT(t, int64(3), *prop.MinItems, "'time_slice' should have had 3 min items")
	assert.EqualT(t, int64(10), *prop.MaxItems, "'time_slice' should have had 10 max items")

	scantest.AssertArrayProperty(t, &schema, "array", "bar_slice", "", "BarSlice")
	prop, ok = schema.Properties["bar_slice"]
	assert.EqualT(t, "a BarSlice has bars which are strings", prop.Description)
	assert.TrueT(t, ok, "should have a 'bar_slice' property")
	require.NotNil(t, prop.Items, "bar_slice should have had an items property")
	require.NotNil(t, prop.Items.Schema, "bar_slice.items should have had a schema property")
	assert.TrueT(t, prop.UniqueItems, "'bar_slice' should have unique items")
	assert.EqualT(t, int64(3), *prop.MinItems, "'bar_slice' should have had 3 min items")
	assert.EqualT(t, int64(10), *prop.MaxItems, "'bar_slice' should have had 10 max items")

	itprop = prop.Items.Schema
	require.NotNil(t, itprop)
	assert.EqualT(t, int64(4), *itprop.MinItems, "'bar_slice.items.minItems' should have been 4")
	assert.EqualT(t, int64(9), *itprop.MaxItems, "'bar_slice.items.maxItems' should have been 9")

	itprop2 := itprop.Items.Schema
	require.NotNil(t, itprop2)
	assert.EqualT(t, int64(5), *itprop2.MinItems, "'bar_slice.items.items.minItems' should have been 5")
	assert.EqualT(t, int64(8), *itprop2.MaxItems, "'bar_slice.items.items.maxItems' should have been 8")

	itprop3 := itprop2.Items.Schema
	require.NotNil(t, itprop3)
	assert.EqualT(t, int64(3), *itprop3.MinLength, "'bar_slice.items.items.items.minLength' should have been 3")
	assert.EqualT(t, int64(10), *itprop3.MaxLength, "'bar_slice.items.items.items.maxLength' should have been 10")
	assert.EqualT(t, "\\w+", itprop3.Pattern, "'bar_slice.items.items.items.pattern' should have \\w+")

	scantest.AssertArrayProperty(t, &schema, "array", "deep_time_slice", "", "DeepTimeSlice")
	prop, ok = schema.Properties["deep_time_slice"]
	assert.EqualT(t, "a DeepSlice has bars which are time", prop.Description)
	assert.TrueT(t, ok, "should have a 'deep_time_slice' property")
	require.NotNil(t, prop.Items, "deep_time_slice should have had an items property")
	require.NotNil(t, prop.Items.Schema, "deep_time_slice.items should have had a schema property")
	assert.TrueT(t, prop.UniqueItems, "'deep_time_slice' should have unique items")
	assert.EqualT(t, int64(3), *prop.MinItems, "'deep_time_slice' should have had 3 min items")
	assert.EqualT(t, int64(10), *prop.MaxItems, "'deep_time_slice' should have had 10 max items")
	itprop = prop.Items.Schema
	require.NotNil(t, itprop)
	assert.EqualT(t, int64(4), *itprop.MinItems, "'deep_time_slice.items.minItems' should have been 4")
	assert.EqualT(t, int64(9), *itprop.MaxItems, "'deep_time_slice.items.maxItems' should have been 9")

	itprop2 = itprop.Items.Schema
	require.NotNil(t, itprop2)
	assert.EqualT(t, int64(5), *itprop2.MinItems, "'deep_time_slice.items.items.minItems' should have been 5")
	assert.EqualT(t, int64(8), *itprop2.MaxItems, "'deep_time_slice.items.items.maxItems' should have been 8")

	itprop3 = itprop2.Items.Schema
	require.NotNil(t, itprop3)

	scantest.AssertArrayProperty(t, &schema, "object", "items", "", "Items")
	prop, ok = schema.Properties["items"]
	assert.TrueT(t, ok, "should have an 'items' slice")
	assert.NotNil(t, prop.Items, "items should have had an items property")
	assert.NotNil(t, prop.Items.Schema, "items.items should have had a schema property")
	itprop = prop.Items.Schema
	assert.Len(t, itprop.Properties, 5)
	assert.Len(t, itprop.Required, 4)
	scantest.AssertProperty(t, itprop, "integer", "id", "int32", "ID")
	iprop, ok := itprop.Properties["id"]
	assert.TrueT(t, ok)
	assert.EqualT(t,
		"ID of this no model instance.\nids in this application start at 11 and are smaller than 1000",
		iprop.Description,
	)
	require.NotNil(t, iprop.Maximum)
	assert.InDeltaT(t, 1000.00, *iprop.Maximum, epsilon)
	assert.TrueT(t, iprop.ExclusiveMaximum, "'id' should have had an exclusive maximum")
	require.NotNil(t, iprop.Minimum)
	assert.InDeltaT(t, 10.00, *iprop.Minimum, epsilon)
	assert.TrueT(t, iprop.ExclusiveMinimum, "'id' should have had an exclusive minimum")
	assert.Equal(t, 11, iprop.Default, "ID default value is incorrect")

	scantest.AssertRef(t, itprop, "pet", "Pet", "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/pet")
	iprop, ok = itprop.Properties["pet"]
	assert.TrueT(t, ok)
	if itprop.Ref.String() != "" {
		assert.EqualT(t, "The Pet to add to this NoModel items bucket.\nPets can appear more than once in the bucket",
			iprop.Description,
		)
	}

	scantest.AssertProperty(t, itprop, "integer", "quantity", "int16", "Quantity")
	iprop, ok = itprop.Properties["quantity"]
	assert.TrueT(t, ok)
	assert.EqualT(t, "The amount of pets to add to this bucket.", iprop.Description)
	assert.InDeltaT(t, 1.00, *iprop.Minimum, epsilon)
	assert.InDeltaT(t, 10.00, *iprop.Maximum, epsilon)

	scantest.AssertProperty(t, itprop, "string", "expiration", "date-time", "Expiration")
	iprop, ok = itprop.Properties["expiration"]
	assert.TrueT(t, ok)
	assert.EqualT(t, "A dummy expiration date.", iprop.Description)

	scantest.AssertProperty(t, itprop, "string", "notes", "", "Notes")
	iprop, ok = itprop.Properties["notes"]
	assert.TrueT(t, ok)
	assert.EqualT(t, "Notes to add to this item.\nThis can be used to add special instructions.", iprop.Description)

	decl2 := getClassificationModel(ctx, "StoreOrder")
	require.NotNil(t, decl2)
	require.NoError(t, NewBuilder(ctx, decl2).Build(WithDefinitions(models)))
	msch, ok := models[scantest.ResolveTestKey(t, models, "order")]
	pn := fixturesModule + "/goparsing/classification/models"
	assert.TrueT(t, ok)
	assert.Equal(t, pn, msch.Extensions["x-go-package"])
	assert.Equal(t, "StoreOrder", msch.Extensions["x-go-name"])
}

func TestBuilder_AddExtensions(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	models := make(map[string]oaispec.Schema)
	decl := getClassificationModel(ctx, "StoreOrder")
	require.NotNil(t, decl)
	require.NoError(t, NewBuilder(ctx, decl).Build(WithDefinitions(models)))

	msch, ok := models[scantest.ResolveTestKey(t, models, "order")]
	pn := fixturesModule + "/goparsing/classification/models"
	assert.TrueT(t, ok)
	assert.Equal(t, pn, msch.Extensions["x-go-package"])
	assert.Equal(t, "StoreOrder", msch.Extensions["x-go-name"])
	assert.EqualT(t, "StoreOrder represents an order in this application.", msch.Title)
}

func TestTextMarhalCustomType(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "TextMarshalModel")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "TextMarshalModel")]
	scantest.AssertProperty(t, &schema, "string", "id", "uuid", "ID")
	scantest.AssertArrayProperty(t, &schema, "string", "ids", "uuid", "IDs")
	scantest.AssertProperty(t, &schema, "string", "struct", "", "Struct")
	scantest.AssertProperty(t, &schema, "string", "map", "", "Map")
	assertMapProperty(t, &schema, "string", "mapUUID", "uuid", "MapUUID")
	urlRef := "#/definitions/" + fixturesModule + "/goparsing/classification/models/URL"
	scantest.AssertRef(t, &schema, "url", "URL", urlRef)
	scantest.AssertProperty(t, &schema, "string", "time", "date-time", "Time")
	scantest.AssertProperty(t, &schema, "string", "structStrfmt", "date-time", "StructStrfmt")
	scantest.AssertProperty(t, &schema, "string", "structStrfmtPtr", "date-time", "StructStrfmtPtr")
	scantest.AssertRef(t, &schema, "customUrl", "CustomURL", urlRef)
}

func TestEmbeddedTypes(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "ComplexerOne")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "ComplexerOne")]
	scantest.AssertProperty(t, &schema, "integer", "age", "int32", "Age")
	scantest.AssertProperty(t, &schema, "integer", "id", "int64", "ID")
	scantest.AssertProperty(t, &schema, "string", "createdAt", "date-time", "CreatedAt")
	scantest.AssertProperty(t, &schema, "string", "extra", "", "Extra")
	scantest.AssertProperty(t, &schema, "string", "name", "", "Name")
	scantest.AssertProperty(t, &schema, "string", "notes", "", "Notes")
}

func TestParsePrimitiveSchemaProperty(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "PrimateModel")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "PrimateModel")]
	scantest.AssertProperty(t, &schema, "boolean", "a", "", "A")
	scantest.AssertProperty(t, &schema, "integer", "b", "int32", "B")
	scantest.AssertProperty(t, &schema, "string", "c", "", "C")
	scantest.AssertProperty(t, &schema, "integer", "d", "int64", "D")
	scantest.AssertProperty(t, &schema, "integer", "e", "int8", "E")
	scantest.AssertProperty(t, &schema, "integer", "f", "int16", "F")
	scantest.AssertProperty(t, &schema, "integer", "g", "int32", "G")
	scantest.AssertProperty(t, &schema, "integer", "h", "int64", "H")
	scantest.AssertProperty(t, &schema, "integer", "i", "uint64", "I")
	scantest.AssertProperty(t, &schema, "integer", "j", "uint8", "J")
	scantest.AssertProperty(t, &schema, "integer", "k", "uint16", "K")
	scantest.AssertProperty(t, &schema, "integer", "l", "uint32", "L")
	scantest.AssertProperty(t, &schema, "integer", "m", "uint64", "M")
	scantest.AssertProperty(t, &schema, "number", "n", "float", "N")
	scantest.AssertProperty(t, &schema, "number", "o", "double", "O")
	scantest.AssertProperty(t, &schema, "integer", "p", "uint8", "P")
	scantest.AssertProperty(t, &schema, "integer", "q", "uint64", "Q")
}

func TestParseStringFormatSchemaProperty(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "FormattedModel")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "FormattedModel")]
	scantest.AssertProperty(t, &schema, "string", "a", "byte", "A")
	scantest.AssertProperty(t, &schema, "string", "b", "creditcard", "B")
	scantest.AssertProperty(t, &schema, "string", "c", "date", "C")
	scantest.AssertProperty(t, &schema, "string", "d", "date-time", "D")
	scantest.AssertProperty(t, &schema, "string", "e", "duration", "E")
	scantest.AssertProperty(t, &schema, "string", "f", "email", "F")
	scantest.AssertProperty(t, &schema, "string", "g", "hexcolor", "G")
	scantest.AssertProperty(t, &schema, "string", "h", "hostname", "H")
	scantest.AssertProperty(t, &schema, "string", "i", "ipv4", "I")
	scantest.AssertProperty(t, &schema, "string", "j", "ipv6", "J")
	scantest.AssertProperty(t, &schema, "string", "k", "isbn", "K")
	scantest.AssertProperty(t, &schema, "string", "l", "isbn10", "L")
	scantest.AssertProperty(t, &schema, "string", "m", "isbn13", "M")
	scantest.AssertProperty(t, &schema, "string", "n", "rgbcolor", "N")
	scantest.AssertProperty(t, &schema, "string", "o", "ssn", "O")
	scantest.AssertProperty(t, &schema, "string", "p", "uri", "P")
	scantest.AssertProperty(t, &schema, "string", "q", "uuid", "Q")
	scantest.AssertProperty(t, &schema, "string", "r", "uuid3", "R")
	scantest.AssertProperty(t, &schema, "string", "s", "uuid4", "S")
	scantest.AssertProperty(t, &schema, "string", "t", "uuid5", "T")
	scantest.AssertProperty(t, &schema, "string", "u", "mac", "U")
}

func TestStringStructTag(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "JSONString")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	sch := models[scantest.ResolveTestKey(t, models, "jsonString")]
	scantest.AssertProperty(t, &sch, "string", "someInt", "int64", "SomeInt")
	scantest.AssertProperty(t, &sch, "string", "someInt8", "int8", "SomeInt8")
	scantest.AssertProperty(t, &sch, "string", "someInt16", "int16", "SomeInt16")
	scantest.AssertProperty(t, &sch, "string", "someInt32", "int32", "SomeInt32")
	scantest.AssertProperty(t, &sch, "string", "someInt64", "int64", "SomeInt64")
	scantest.AssertProperty(t, &sch, "string", "someUint", "uint64", "SomeUint")
	scantest.AssertProperty(t, &sch, "string", "someUint8", "uint8", "SomeUint8")
	scantest.AssertProperty(t, &sch, "string", "someUint16", "uint16", "SomeUint16")
	scantest.AssertProperty(t, &sch, "string", "someUint32", "uint32", "SomeUint32")
	scantest.AssertProperty(t, &sch, "string", "someUint64", "uint64", "SomeUint64")
	scantest.AssertProperty(t, &sch, "string", "someFloat64", "double", "SomeFloat64")
	scantest.AssertProperty(t, &sch, "string", "someString", "", "SomeString")
	scantest.AssertProperty(t, &sch, "string", "someBool", "", "SomeBool")
	scantest.AssertProperty(t, &sch, "string", "SomeDefaultInt", "int64", "")

	prop, ok := sch.Properties["somethingElse"]
	if assert.TrueT(t, ok) {
		assert.NotEqual(t, "string", prop.Type)
	}
}

func TestPtrFieldStringStructTag(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "JSONPtrString")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	sch := models[scantest.ResolveTestKey(t, models, "jsonPtrString")]
	scantest.AssertProperty(t, &sch, "string", "someInt", "int64", "SomeInt")
	scantest.AssertProperty(t, &sch, "string", "someInt8", "int8", "SomeInt8")
	scantest.AssertProperty(t, &sch, "string", "someInt16", "int16", "SomeInt16")
	scantest.AssertProperty(t, &sch, "string", "someInt32", "int32", "SomeInt32")
	scantest.AssertProperty(t, &sch, "string", "someInt64", "int64", "SomeInt64")
	scantest.AssertProperty(t, &sch, "string", "someUint", "uint64", "SomeUint")
	scantest.AssertProperty(t, &sch, "string", "someUint8", "uint8", "SomeUint8")
	scantest.AssertProperty(t, &sch, "string", "someUint16", "uint16", "SomeUint16")
	scantest.AssertProperty(t, &sch, "string", "someUint32", "uint32", "SomeUint32")
	scantest.AssertProperty(t, &sch, "string", "someUint64", "uint64", "SomeUint64")
	scantest.AssertProperty(t, &sch, "string", "someFloat64", "double", "SomeFloat64")
	scantest.AssertProperty(t, &sch, "string", "someString", "", "SomeString")
	scantest.AssertProperty(t, &sch, "string", "someBool", "", "SomeBool")

	prop, ok := sch.Properties["somethingElse"]
	if assert.TrueT(t, ok) {
		assert.NotEqual(t, "string", prop.Type)
	}
}

func TestIgnoredStructField(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "IgnoredFields")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	sch := models[scantest.ResolveTestKey(t, models, "ignoredFields")]
	scantest.AssertProperty(t, &sch, "string", "someIncludedField", "", "SomeIncludedField")
	scantest.AssertProperty(t, &sch, "string", "someErroneouslyIncludedField", "", "SomeErroneouslyIncludedField")
	assert.Len(t, sch.Properties, 2)
}

func TestParseStructFields(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "SimpleComplexModel")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "SimpleComplexModel")]
	scantest.AssertProperty(t, &schema, "object", "emb", "", "Emb")
	eSchema := schema.Properties["emb"]
	scantest.AssertProperty(t, &eSchema, "integer", "cid", "int64", "CID")
	scantest.AssertProperty(t, &eSchema, "string", "baz", "", "Baz")

	scantest.AssertRef(t, &schema, "top", "Top", "#/definitions/"+fixturesModule+"/goparsing/classification/models/Something")
	scantest.AssertRef(t, &schema, "notSel", "NotSel", "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/NotSelected")
}

func TestParsePointerFields(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "Pointdexter")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "Pointdexter")]

	scantest.AssertProperty(t, &schema, "integer", "id", "int64", "ID")
	scantest.AssertProperty(t, &schema, "string", "name", "", "Name")
	scantest.AssertProperty(t, &schema, "object", "emb", "", "Emb")
	scantest.AssertProperty(t, &schema, "string", "t", "uuid5", "T")
	eSchema := schema.Properties["emb"]
	scantest.AssertProperty(t, &eSchema, "integer", "cid", "int64", "CID")
	scantest.AssertProperty(t, &eSchema, "string", "baz", "", "Baz")

	scantest.AssertRef(t, &schema, "top", "Top", "#/definitions/"+fixturesModule+"/goparsing/classification/models/Something")
	scantest.AssertRef(t, &schema, "notSel", "NotSel", "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/NotSelected")
}

func TestEmbeddedStarExpr(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "EmbeddedStarExpr")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "EmbeddedStarExpr")]

	scantest.AssertProperty(t, &schema, "integer", "embeddedMember", "int64", "EmbeddedMember")
	scantest.AssertProperty(t, &schema, "integer", "notEmbedded", "int64", "NotEmbedded")
}

func TestArrayOfPointers(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "Cars")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "cars")]
	scantest.AssertProperty(t, &schema, "array", "cars", "", "Cars")
}

func TestOverridingOneIgnore(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "OverridingOneIgnore")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "OverridingOneIgnore")]

	scantest.AssertProperty(t, &schema, "integer", "id", "int64", "ID")
	scantest.AssertProperty(t, &schema, "string", "name", "", "Name")
	assert.Len(t, schema.Properties, 2)
}

type collectionAssertions struct {
	assertProperty func(t *testing.T, schema *oaispec.Schema, typeName, jsonName, format, goName string)
	assertRef      func(t *testing.T, schema *oaispec.Schema, jsonName, goName, fragment string)
	nestedSchema   func(prop oaispec.Schema) *oaispec.Schema
}

func testParseCollectionFields(
	t *testing.T,
	modelName string,
	ca collectionAssertions,
) {
	t.Helper()
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, modelName)
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, modelName)]

	ca.assertProperty(t, &schema, "integer", "ids", "int64", "IDs")
	ca.assertProperty(t, &schema, "string", "names", "", "Names")
	ca.assertProperty(t, &schema, "string", "uuids", "uuid", "UUIDs")
	ca.assertProperty(t, &schema, "object", "embs", "", "Embs")
	eSchema := ca.nestedSchema(schema.Properties["embs"])
	ca.assertProperty(t, eSchema, "integer", "cid", "int64", "CID")
	ca.assertProperty(t, eSchema, "string", "baz", "", "Baz")

	ca.assertRef(t, &schema, "tops", "Tops", "#/definitions/"+fixturesModule+"/goparsing/classification/models/Something")
	ca.assertRef(t, &schema, "notSels", "NotSels", "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/NotSelected")

	ca.assertProperty(t, &schema, "integer", "ptrIds", "int64", "PtrIDs")
	ca.assertProperty(t, &schema, "string", "ptrNames", "", "PtrNames")
	ca.assertProperty(t, &schema, "string", "ptrUuids", "uuid", "PtrUUIDs")
	ca.assertProperty(t, &schema, "object", "ptrEmbs", "", "PtrEmbs")
	eSchema = ca.nestedSchema(schema.Properties["ptrEmbs"])
	ca.assertProperty(t, eSchema, "integer", "ptrCid", "int64", "PtrCID")
	ca.assertProperty(t, eSchema, "string", "ptrBaz", "", "PtrBaz")

	ca.assertRef(t, &schema, "ptrTops", "PtrTops", "#/definitions/"+fixturesModule+"/goparsing/classification/models/Something")
	ca.assertRef(t, &schema, "ptrNotSels", "PtrNotSels", "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/NotSelected")
}

func TestParseSliceFields(t *testing.T) {
	testParseCollectionFields(t, "SliceAndDice", collectionAssertions{
		assertProperty: scantest.AssertArrayProperty,
		assertRef:      assertArrayRef,
		nestedSchema:   func(prop oaispec.Schema) *oaispec.Schema { return prop.Items.Schema },
	})
}

func TestParseMapFields(t *testing.T) {
	testParseCollectionFields(t, "MapTastic", collectionAssertions{
		assertProperty: assertMapProperty,
		assertRef:      assertMapRef,
		nestedSchema:   func(prop oaispec.Schema) *oaispec.Schema { return prop.AdditionalProperties.Schema },
	})
}

func TestInterfaceField(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "Interfaced")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "Interfaced")]
	scantest.AssertProperty(t, &schema, "", "custom_data", "", "CustomData")
}

func TestAliasedTypes(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "OtherTypes")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "OtherTypes")]
	// Sub-builder unit tests run without the spec reduce stage, so $refs
	// stay fully-qualified. The alias targets are not built into the local
	// map (only OtherTypes is), so ResolveTestKey cannot shorten them;
	// hardcode their package-qualified keys.
	mp := "#/definitions/" + fixturesModule + "/goparsing/classification/models/"
	tp := "#/definitions/" + fixturesModule + "/goparsing/classification/transitive/mods/"
	scantest.AssertRef(t, &schema, "named", "Named", mp+"SomeStringType")
	scantest.AssertRef(t, &schema, "numbered", "Numbered", mp+"SomeIntType")
	scantest.AssertProperty(t, &schema, "string", "dated", "date-time", "Dated")
	scantest.AssertRef(t, &schema, "timed", "Timed", mp+"SomeTimedType")
	scantest.AssertRef(t, &schema, "petted", "Petted", mp+"SomePettedType")
	scantest.AssertRef(t, &schema, "somethinged", "Somethinged", mp+"SomethingType")
	scantest.AssertRef(t, &schema, "strMap", "StrMap", mp+"SomeStringMap")
	scantest.AssertRef(t, &schema, "strArrMap", "StrArrMap", mp+"SomeArrayStringMap")

	scantest.AssertRef(t, &schema, "manyNamed", "ManyNamed", mp+"SomeStringsType")
	scantest.AssertRef(t, &schema, "manyNumbered", "ManyNumbered", mp+"SomeIntsType")
	scantest.AssertArrayProperty(t, &schema, "string", "manyDated", "date-time", "ManyDated")
	scantest.AssertRef(t, &schema, "manyTimed", "ManyTimed", mp+"SomeTimedsType")
	scantest.AssertRef(t, &schema, "manyPetted", "ManyPetted", mp+"SomePettedsType")
	scantest.AssertRef(t, &schema, "manySomethinged", "ManySomethinged", mp+"SomethingsType")

	assertArrayRef(t, &schema, "nameds", "Nameds", mp+"SomeStringType")
	assertArrayRef(t, &schema, "numbereds", "Numbereds", mp+"SomeIntType")
	scantest.AssertArrayProperty(t, &schema, "string", "dateds", "date-time", "Dateds")
	assertArrayRef(t, &schema, "timeds", "Timeds", mp+"SomeTimedType")
	assertArrayRef(t, &schema, "petteds", "Petteds", mp+"SomePettedType")
	assertArrayRef(t, &schema, "somethingeds", "Somethingeds", mp+"SomethingType")

	scantest.AssertRef(t, &schema, "modsNamed", "ModsNamed", tp+"modsSomeStringType")
	scantest.AssertRef(t, &schema, "modsNumbered", "ModsNumbered", tp+"modsSomeIntType")
	// F1: modsSomeTimeType is swagger:model + swagger:strfmt date-time, so it
	// now $refs its definition (which carries the format) like its siblings,
	// rather than inlining {string,date-time}.
	scantest.AssertRef(t, &schema, "modsDated", "ModsDated", tp+"modsSomeTimeType")
	scantest.AssertRef(t, &schema, "modsTimed", "ModsTimed", tp+"modsSomeTimedType")
	scantest.AssertRef(t, &schema, "modsPetted", "ModsPetted", tp+"modsSomePettedType")

	assertArrayRef(t, &schema, "modsNameds", "ModsNameds", tp+"modsSomeStringType")
	assertArrayRef(t, &schema, "modsNumbereds", "ModsNumbereds", tp+"modsSomeIntType")
	assertArrayRef(t, &schema, "modsDateds", "ModsDateds", tp+"modsSomeTimeType")
	assertArrayRef(t, &schema, "modsTimeds", "ModsTimeds", tp+"modsSomeTimedType")
	assertArrayRef(t, &schema, "modsPetteds", "ModsPetteds", tp+"modsSomePettedType")

	scantest.AssertRef(t, &schema, "manyModsNamed", "ManyModsNamed", tp+"modsSomeStringsType")
	scantest.AssertRef(t, &schema, "manyModsNumbered", "ManyModsNumbered", tp+"modsSomeIntsType")
	// F1: modsSomeTimesType (model + strfmt date-time) now $refs its definition.
	scantest.AssertRef(t, &schema, "manyModsDated", "ManyModsDated", tp+"modsSomeTimesType")
	scantest.AssertRef(t, &schema, "manyModsTimed", "ManyModsTimed", tp+"modsSomeTimedsType")
	scantest.AssertRef(t, &schema, "manyModsPetted", "ManyModsPetted", tp+"modsSomePettedsType")
	scantest.AssertRef(t, &schema, "manyModsPettedPtr", "ManyModsPettedPtr", tp+"modsSomePettedsPtrType")

	// swagger:alias is deprecated (F8): it no longer force-inlines the
	// primitive, so these now $ref their definitions like any other named
	// type (consistent with the `named`/`numbered` assertions above).
	scantest.AssertRef(t, &schema, "namedAlias", "NamedAlias", mp+"SomeStringTypeAlias")
	scantest.AssertRef(t, &schema, "numberedAlias", "NumberedAlias", mp+"SomeIntTypeAlias")
	assertArrayRef(t, &schema, "namedsAlias", "NamedsAlias", mp+"SomeStringTypeAlias")
	assertArrayRef(t, &schema, "numberedsAlias", "NumberedsAlias", mp+"SomeIntTypeAlias")
}

func TestAliasedModels(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)

	names := []string{
		"SomeStringType",
		"SomeIntType",
		"SomeTimeType",
		"SomeTimedType",
		"SomePettedType",
		"SomethingType",
		"SomeStringsType",
		"SomeIntsType",
		"SomeTimesType",
		"SomeTimedsType",
		"SomePettedsType",
		"SomethingsType",
		"SomeObject",
		"SomeStringMap",
		"SomeIntMap",
		"SomeTimeMap",
		"SomeTimedMap",
		"SomePettedMap",
		"SomeSomethingMap",
	}

	defs := make(map[string]oaispec.Schema)
	for _, nm := range names {
		decl := getClassificationModel(ctx, nm)
		require.NotNil(t, decl)

		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(WithDefinitions(defs)))
	}

	for k := range defs {
		for i, b := range names {
			// defs is keyed by the fully-qualified identity; match on the
			// leaf via ResolveTestKey rather than the bare name.
			if scantest.ResolveTestKey(t, defs, b) == k {
				// remove the entry from the collection
				names = append(names[:i], names[i+1:]...)
			}
		}
	}
	// Sub-builder unit tests run without the spec reduce stage; the pet /
	// Something $ref targets are not in the local map, so their keys are
	// hardcoded with their package paths.
	petRef := "#/definitions/" + fixturesModule + "/goparsing/classification/transitive/mods/pet"
	somethingRef := "#/definitions/" + fixturesModule + "/goparsing/classification/models/Something"
	if assert.Empty(t, names) {
		// single value types
		assertDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeStringType"), "string", "")
		assertDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeIntType"), "integer", "int64")
		assertDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeTimeType"), "string", "date-time")
		assertDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeTimedType"), "string", "date-time")
		assertRefDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomePettedType"), petRef, "")
		assertRefDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomethingType"), somethingRef, "")

		// slice types
		assertArrayDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeStringsType"), "string", "", "")
		assertArrayDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeIntsType"), "integer", "int64", "")
		assertArrayDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeTimesType"), "string", "date-time", "")
		assertArrayDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeTimedsType"), "string", "date-time", "")
		assertArrayWithRefDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomePettedsType"), petRef, "")
		assertArrayWithRefDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomethingsType"), somethingRef, "")

		// map types
		assertMapDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeObject"), "object", "", "")
		assertMapDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeStringMap"), "string", "", "")
		assertMapDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeIntMap"), "integer", "int64", "")
		assertMapDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeTimeMap"), "string", "date-time", "")
		assertMapDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeTimedMap"), "string", "date-time", "")
		assertMapWithRefDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomePettedMap"), petRef, "")
		assertMapWithRefDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeSomethingMap"), somethingRef, "")
	}
}

func TestAliasedTopLevelModels(t *testing.T) {
	t.Run("with options: no scan models, with aliases as ref", func(t *testing.T) {
		t.Run("with goparsing/spec", func(t *testing.T) {
			ctx, err := scanner.NewScanCtx(&scanner.Options{
				Packages: []string{
					"./goparsing/spec",
				},
				WorkDir:    scantest.FixturesDir(),
				ScanModels: false,
				RefAliases: true,
			})
			require.NoError(t, err)

			t.Run("should find User definition in source", func(t *testing.T) {
				_, hasUser := ctx.FindDecl(fixturesModule+"/goparsing/spec", "User")
				require.TrueT(t, hasUser)
			})

			var decl *scanner.EntityDecl
			t.Run("should find Customer definition in source", func(t *testing.T) {
				var hasCustomer bool
				decl, hasCustomer = ctx.FindDecl(fixturesModule+"/goparsing/spec", "Customer")
				require.TrueT(t, hasCustomer)
			})

			t.Run("with schema builder", func(t *testing.T) {
				require.NotNil(t, decl)
				builder := NewBuilder(ctx, decl)

				t.Run("should build model for Customer", func(t *testing.T) {
					models := make(map[string]oaispec.Schema)
					require.NoError(t, builder.Build(WithDefinitions(models)))

					assertRefDefinition(t, models, scantest.ResolveTestKey(t, models, "Customer"),
						"#/definitions/"+fixturesModule+"/goparsing/spec/User", "")
				})

				t.Run("should have discovered models for User and Customer", func(t *testing.T) {
					require.Len(t, builder.PostDeclarations(), 2)
					foundUserIndex := -1
					foundCustomerIndex := -1

					for i, discoveredDecl := range builder.PostDeclarations() {
						switch discoveredDecl.Obj().Name() {
						case "User":
							foundUserIndex = i
						case "Customer":
							foundCustomerIndex = i
						}
					}
					require.GreaterOrEqualT(t, foundUserIndex, 0)
					require.GreaterOrEqualT(t, foundCustomerIndex, 0)
					postDecls := builder.PostDeclarations()
					require.GreaterT(t, len(postDecls), foundUserIndex)

					userBuilder := NewBuilder(ctx, postDecls[foundUserIndex])

					t.Run("should build model for User", func(t *testing.T) {
						models := make(map[string]oaispec.Schema)
						require.NoError(t, userBuilder.Build(WithDefinitions(models)))

						require.MapContainsT(t, models, scantest.ResolveTestKey(t, models, "User"))

						user := models[scantest.ResolveTestKey(t, models, "User")]
						assert.TrueT(t, user.Type.Contains("object"))

						userProperties := user.Properties
						require.MapContainsT(t, userProperties, "name")
					})
				})
			})
		})
	})

	t.Run("with options: no scan models, without aliases as ref", func(t *testing.T) {
		t.Run("with goparsing/spec", func(t *testing.T) {
			ctx, err := scanner.NewScanCtx(&scanner.Options{
				Packages: []string{
					"./goparsing/spec",
				},
				WorkDir:    scantest.FixturesDir(),
				ScanModels: false,
				RefAliases: false,
			})
			require.NoError(t, err)

			t.Run("should find User definition in source", func(t *testing.T) {
				_, hasUser := ctx.FindDecl(fixturesModule+"/goparsing/spec", "User")
				require.TrueT(t, hasUser)
			})

			var decl *scanner.EntityDecl
			t.Run("should find Customer definition in source", func(t *testing.T) {
				var hasCustomer bool
				decl, hasCustomer = ctx.FindDecl(fixturesModule+"/goparsing/spec", "Customer")
				require.TrueT(t, hasCustomer)
			})

			t.Run("with schema builder", func(t *testing.T) {
				require.NotNil(t, decl)
				builder := NewBuilder(ctx, decl)

				t.Run("should build model for Customer", func(t *testing.T) {
					models := make(map[string]oaispec.Schema)
					require.NoError(t, builder.Build(WithDefinitions(models)))

					require.MapContainsT(t, models, scantest.ResolveTestKey(t, models, "Customer"))
					customer := models[scantest.ResolveTestKey(t, models, "Customer")]
					require.MapNotContainsT(t, models, scantest.ResolveTestKey(t, models, "User"))

					assert.TrueT(t, customer.Type.Contains("object"))

					customerProperties := customer.Properties
					assert.MapContainsT(t, customerProperties, "name")
					assert.NotEmpty(t, customer.Title)
				})

				t.Run("should have discovered only Customer", func(t *testing.T) {
					postDecls := builder.PostDeclarations()
					require.Len(t, postDecls, 1)
					discovered := postDecls[0]
					assert.EqualT(t, "Customer", discovered.Obj().Name())
				})
			})
		})
	})
}

func TestEmbeddedAllOf(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "AllOfModel")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "AllOfModel")]

	require.Len(t, schema.AllOf, 3)
	asch := schema.AllOf[0]
	scantest.AssertProperty(t, &asch, "integer", "age", "int32", "Age")
	scantest.AssertProperty(t, &asch, "integer", "id", "int64", "ID")
	scantest.AssertProperty(t, &asch, "string", "name", "", "Name")

	asch = schema.AllOf[1]
	assert.EqualT(t, "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/withNotes", asch.Ref.String())

	asch = schema.AllOf[2]
	scantest.AssertProperty(t, &asch, "string", "createdAt", "date-time", "CreatedAt")
	scantest.AssertProperty(t, &asch, "integer", "did", "int64", "DID")
	scantest.AssertProperty(t, &asch, "string", "cat", "", "Cat")
}

func TestPointersAreNullableByDefaultWhenSetXNullableForPointersIsSet(t *testing.T) {
	allModels := make(map[string]oaispec.Schema)
	assertModel := func(ctx *scanner.ScanCtx, packagePath, modelName string) {
		decl, _ := ctx.FindDecl(packagePath, modelName)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(WithDefinitions(allModels)))

		schema := allModels[scantest.ResolveTestKey(t, allModels, modelName)]
		require.Len(t, schema.Properties, 5)

		// Interface-method properties are camelCased; struct fields
		// without json tags keep the Go identifier verbatim.
		v1, v2, v3, v4, v5 := valueKeys(modelName)

		require.MapContainsT(t, schema.Properties, v1)
		assert.Equal(t, true, schema.Properties[v1].Extensions["x-nullable"])
		require.MapContainsT(t, schema.Properties, v2)
		assert.MapNotContainsT(t, schema.Properties[v2].Extensions, "x-nullable")
		require.MapContainsT(t, schema.Properties, v3)
		assert.Equal(t, false, schema.Properties[v3].Extensions["x-nullable"])
		require.MapContainsT(t, schema.Properties, v4)
		assert.MapNotContainsT(t, schema.Properties[v4].Extensions, "x-nullable")
		assert.Equal(t, false, schema.Properties[v4].Extensions["x-isnullable"])
		require.MapContainsT(t, schema.Properties, v5)
		assert.MapNotContainsT(t, schema.Properties[v5].Extensions, "x-nullable")
	}

	packagePattern := "./enhancements/pointers-nullable-by-default"
	packagePath := fixturesModule + "/enhancements/pointers-nullable-by-default"
	ctx, err := scanner.NewScanCtx(&scanner.Options{Packages: []string{packagePattern}, WorkDir: scantest.FixturesDir(), SetXNullableForPointers: true})
	require.NoError(t, err)

	assertModel(ctx, packagePath, "Item")
	assertModel(ctx, packagePath, "ItemInterface")
}

// valueKeys returns the five property keys expected for the fixtures
// Item (struct, Go names verbatim) and ItemInterface (interface
// methods, JSON-name-derived via the interface-method mangler — see
// [§method-mangler](./README.md#method-mangler) — so the keys are
// camelCased rather than Go-verbatim).
func valueKeys(modelName string) (string, string, string, string, string) {
	if modelName == "ItemInterface" {
		return sampleValue1, sampleValue2, "value3", "value4", "value5"
	}
	return "Value1", "Value2", "Value3", "Value4", "Value5"
}

func TestPointersAreNotNullableByDefaultWhenSetXNullableForPointersIsNotSet(t *testing.T) {
	allModels := make(map[string]oaispec.Schema)
	assertModel := func(ctx *scanner.ScanCtx, packagePath, modelName string) {
		decl, _ := ctx.FindDecl(packagePath, modelName)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(WithDefinitions(allModels)))

		schema := allModels[scantest.ResolveTestKey(t, allModels, modelName)]
		require.Len(t, schema.Properties, 5)

		v1, v2, v3, v4, v5 := valueKeys(modelName)

		require.MapContainsT(t, schema.Properties, v1)
		assert.MapNotContainsT(t, schema.Properties[v1].Extensions, "x-nullable")
		require.MapContainsT(t, schema.Properties, v2)
		assert.MapNotContainsT(t, schema.Properties[v2].Extensions, "x-nullable")
		require.MapContainsT(t, schema.Properties, v3)
		assert.Equal(t, false, schema.Properties[v3].Extensions["x-nullable"])
		require.MapContainsT(t, schema.Properties, v4)
		assert.MapNotContainsT(t, schema.Properties[v4].Extensions, "x-nullable")
		assert.Equal(t, false, schema.Properties[v4].Extensions["x-isnullable"])
		require.MapContainsT(t, schema.Properties, v5)
		assert.MapNotContainsT(t, schema.Properties[v5].Extensions, "x-nullable")
	}

	packagePattern := "./enhancements/pointers-nullable-by-default"
	packagePath := fixturesModule + "/enhancements/pointers-nullable-by-default"
	ctx, err := scanner.NewScanCtx(&scanner.Options{Packages: []string{packagePattern}, WorkDir: scantest.FixturesDir()})
	require.NoError(t, err)

	assertModel(ctx, packagePath, "Item")
	assertModel(ctx, packagePath, "ItemInterface")
}

func TestSwaggerTypeNamed(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "NamedWithType")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "namedWithType")]

	scantest.AssertProperty(t, &schema, "object", "some_map", "", "SomeMap")
}

func TestSwaggerTypeNamedWithGenerics(t *testing.T) {
	tests := map[string]func(t *testing.T, models map[string]oaispec.Schema){
		"NamedStringResults": func(t *testing.T, models map[string]oaispec.Schema) {
			schema := models[scantest.ResolveTestKey(t, models, "namedStringResults")]
			scantest.AssertArrayProperty(t, &schema, "string", "matches", "", "Matches")
		},
		"NamedStoreOrderResults": func(t *testing.T, models map[string]oaispec.Schema) {
			schema := models[scantest.ResolveTestKey(t, models, "namedStoreOrderResults")]
			assertArrayRef(t, &schema, "matches", "Matches", "#/definitions/"+fixturesModule+"/goparsing/classification/models/order")
		},
		"NamedStringSlice": func(t *testing.T, models map[string]oaispec.Schema) {
			assertArrayDefinition(t, models, scantest.ResolveTestKey(t, models, "namedStringSlice"), "string", "", "NamedStringSlice")
		},
		"NamedStoreOrderSlice": func(t *testing.T, models map[string]oaispec.Schema) {
			assertArrayWithRefDefinition(t, models, scantest.ResolveTestKey(t, models, "namedStoreOrderSlice"), classificationOrderRef, "NamedStoreOrderSlice")
		},
		"NamedStringMap": func(t *testing.T, models map[string]oaispec.Schema) {
			assertMapDefinition(t, models, scantest.ResolveTestKey(t, models, "namedStringMap"), "string", "", "NamedStringMap")
		},
		"NamedStoreOrderMap": func(t *testing.T, models map[string]oaispec.Schema) {
			assertMapWithRefDefinition(t, models, scantest.ResolveTestKey(t, models, "namedStoreOrderMap"), classificationOrderRef, "NamedStoreOrderMap")
		},
		"NamedMapOfStoreOrderSlices": func(t *testing.T, models map[string]oaispec.Schema) {
			assertMapDefinition(t, models, scantest.ResolveTestKey(t, models, "namedMapOfStoreOrderSlices"), "array", "", "NamedMapOfStoreOrderSlices")
			arraySchema := models[scantest.ResolveTestKey(t, models, "namedMapOfStoreOrderSlices")].AdditionalProperties.Schema
			assertArrayWithRefDefinition(t, map[string]oaispec.Schema{
				"array": *arraySchema,
			}, "array", "#/definitions/"+fixturesModule+"/goparsing/classification/models/order", "")
		},
	}

	for testName, testFunc := range tests {
		t.Run(testName, func(t *testing.T) {
			ctx := scantest.LoadClassificationPkgsCtx(t)
			decl := getClassificationModel(ctx, testName)
			require.NotNil(t, decl)
			prs := NewBuilder(ctx, decl)
			models := make(map[string]oaispec.Schema)
			require.NoError(t, prs.Build(WithDefinitions(models)))
			testFunc(t, models)
		})
	}
}

func TestSwaggerTypeStruct(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "NullString")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "NullString")]

	assert.TrueT(t, schema.Type.Contains("string"))
}

func TestStructDiscriminators(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)

	models := make(map[string]oaispec.Schema)
	for _, tn := range []string{"BaseStruct", "Giraffe", "Gazelle"} {
		decl := getClassificationModel(ctx, tn)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(WithDefinitions(models)))
	}

	schema := models[scantest.ResolveTestKey(t, models, "animal")]

	assert.Equal(t, "BaseStruct", schema.Extensions["x-go-name"])
	assert.EqualT(t, "jsonClass", schema.Discriminator)

	sch := models[scantest.ResolveTestKey(t, models, "gazelle")]
	assert.Len(t, sch.AllOf, 2)
	cl, _ := sch.Extensions.GetString("x-class")
	assert.EqualT(t, "a.b.c.d.E", cl)
	cl, _ = sch.Extensions.GetString("x-go-name")
	assert.EqualT(t, "Gazelle", cl)

	sch = models[scantest.ResolveTestKey(t, models, "giraffe")]
	assert.Len(t, sch.AllOf, 2)
	cl, _ = sch.Extensions.GetString("x-class")
	assert.Empty(t, cl)
	cl, _ = sch.Extensions.GetString("x-go-name")
	assert.EqualT(t, "Giraffe", cl)

	// sch = noModelDefs["lion"]

	// b, _ := json.MarshalIndent(sch, "", "  ")
	// fmt.Println(string(b))
}

func TestInterfaceDiscriminators(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	models := make(map[string]oaispec.Schema)
	for _, tn := range []string{"BaseStruct", "Identifiable", "WaterType", "Fish", "TeslaCar", "ModelS", "ModelX", "ModelA", "Cars"} {
		decl := getClassificationModel(ctx, tn)
		require.NotNil(t, decl)

		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(WithDefinitions(models)))
	}

	schema, ok := models[scantest.ResolveTestKey(t, models, "fish")]

	if assert.TrueT(t, ok) && assert.Len(t, schema.AllOf, 5) {
		sch := schema.AllOf[3]
		assert.Len(t, sch.Properties, 1)
		scantest.AssertProperty(t, &sch, "string", "colorName", "", "ColorName")

		sch = schema.AllOf[2]
		assert.EqualT(t, "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/extra", sch.Ref.String())

		sch = schema.AllOf[0]
		assert.Len(t, sch.Properties, 1)
		scantest.AssertProperty(t, &sch, "integer", "id", "int64", "ID")

		sch = schema.AllOf[1]
		assert.EqualT(t, "#/definitions/"+fixturesModule+"/goparsing/classification/models/water", sch.Ref.String())

		sch = schema.AllOf[4]
		assert.Len(t, sch.Properties, 2)
		scantest.AssertProperty(t, &sch, "string", "name", "", "Name")
		scantest.AssertProperty(t, &sch, "string", "jsonClass", "", "StructType")
		assert.EqualT(t, "jsonClass", sch.Discriminator)
	}

	schema, ok = models[scantest.ResolveTestKey(t, models, "modelS")]
	if assert.TrueT(t, ok) {
		assert.Len(t, schema.AllOf, 2)
		cl, _ := schema.Extensions.GetString("x-class")
		assert.EqualT(t, "com.tesla.models.ModelS", cl)
		cl, _ = schema.Extensions.GetString("x-go-name")
		assert.EqualT(t, "ModelS", cl)

		sch := schema.AllOf[0]
		assert.EqualT(t, "#/definitions/"+fixturesModule+"/goparsing/classification/models/TeslaCar", sch.Ref.String())
		sch = schema.AllOf[1]
		assert.Len(t, sch.Properties, 1)
		scantest.AssertProperty(t, &sch, "string", "edition", "", "Edition")
	}

	schema, ok = models[scantest.ResolveTestKey(t, models, "modelA")]
	if assert.TrueT(t, ok) {
		cl, _ := schema.Extensions.GetString("x-go-name")
		assert.EqualT(t, "ModelA", cl)

		sch, ok := schema.Properties["Tesla"]
		if assert.TrueT(t, ok) {
			assert.EqualT(t, "#/definitions/"+fixturesModule+"/goparsing/classification/models/TeslaCar", sch.Ref.String())
		}

		scantest.AssertProperty(t, &schema, "integer", "doors", "int64", "Doors")
	}
}

func getClassificationModel(ctx *scanner.ScanCtx, nm string) *scanner.EntityDecl {
	decl, ok := ctx.FindDecl(fixturesModule+"/goparsing/classification/models", nm)
	if !ok {
		return nil
	}
	return decl
}

func assertArrayRef(t *testing.T, schema *oaispec.Schema, jsonName, goName, fragment string) {
	t.Helper()

	scantest.AssertArrayProperty(t, schema, "", jsonName, "", goName)
	psch := schema.Properties[jsonName].Items.Schema
	assert.EqualT(t, fragment, psch.Ref.String())
}

func assertDefinition(t *testing.T, defs map[string]oaispec.Schema, defName, typeName, formatName string) {
	t.Helper()

	schema, ok := defs[defName]
	if assert.TrueT(t, ok) {
		if assert.NotEmpty(t, schema.Type) {
			assert.EqualT(t, typeName, schema.Type[0])
			assert.Nil(t, schema.Extensions["x-go-name"])
			assert.EqualT(t, formatName, schema.Format)
		}
	}
}

func assertMapDefinition(t *testing.T, defs map[string]oaispec.Schema, defName, typeName, formatName, goName string) {
	t.Helper()

	schema, ok := defs[defName]
	require.TrueT(t, ok)
	require.NotEmpty(t, schema.Type)

	assert.EqualT(t, "object", schema.Type[0])
	adl := schema.AdditionalProperties

	require.NotNil(t, adl)
	require.NotNil(t, adl.Schema)

	if len(adl.Schema.Type) > 0 {
		assert.EqualT(t, typeName, adl.Schema.Type[0])
	}
	assert.EqualT(t, formatName, adl.Schema.Format)

	assertExtension(t, schema, goName)
}

func assertExtension(t *testing.T, schema oaispec.Schema, goName string) {
	t.Helper()

	if goName != "" {
		assert.Equal(t, goName, schema.Extensions["x-go-name"])

		return
	}

	assert.Nil(t, schema.Extensions["x-go-name"])
}

func assertMapWithRefDefinition(t *testing.T, defs map[string]oaispec.Schema, defName, refURL, goName string) {
	t.Helper()

	schema, ok := defs[defName]
	require.TrueT(t, ok)
	require.NotEmpty(t, schema.Type)
	assert.EqualT(t, "object", schema.Type[0])
	adl := schema.AdditionalProperties
	require.NotNil(t, adl)
	require.NotNil(t, adl.Schema)
	require.NotZero(t, adl.Schema.Ref)
	assert.EqualT(t, refURL, adl.Schema.Ref.String())
	assertExtension(t, schema, goName)
}

func assertArrayDefinition(t *testing.T, defs map[string]oaispec.Schema, defName, typeName, formatName, goName string) {
	t.Helper()

	schema, ok := defs[defName]
	require.TrueT(t, ok)
	require.NotEmpty(t, schema.Type)
	assert.EqualT(t, "array", schema.Type[0])
	adl := schema.Items
	require.NotNil(t, adl)
	require.NotNil(t, adl.Schema)
	assert.EqualT(t, typeName, adl.Schema.Type[0])
	assert.EqualT(t, formatName, adl.Schema.Format)
	assertExtension(t, schema, goName)
}

func assertArrayWithRefDefinition(t *testing.T, defs map[string]oaispec.Schema, defName, refURL, goName string) {
	t.Helper()

	schema, ok := defs[defName]
	require.TrueT(t, ok)
	require.NotEmpty(t, schema.Type)
	assert.EqualT(t, "array", schema.Type[0])
	adl := schema.Items
	require.NotNil(t, adl)
	require.NotNil(t, adl.Schema)
	require.NotZero(t, adl.Schema.Ref)
	assert.EqualT(t, refURL, adl.Schema.Ref.String())
	assertExtension(t, schema, goName)
}

func assertRefDefinition(t *testing.T, defs map[string]oaispec.Schema, defName, refURL, goName string) {
	schema, ok := defs[defName]
	if assert.TrueT(t, ok) {
		if assert.NotZero(t, schema.Ref) {
			url := schema.Ref.String()
			assert.EqualT(t, refURL, url)
			if goName != "" {
				assert.Equal(t, goName, schema.Extensions["x-go-name"])
			} else {
				assert.Nil(t, schema.Extensions["x-go-name"])
			}
		}
	}
}

func assertMapProperty(t *testing.T, schema *oaispec.Schema, typeName, jsonName, format, goName string) {
	prop := schema.Properties[jsonName]
	assert.NotEmpty(t, prop.Type)
	assert.TrueT(t, prop.Type.Contains("object"))
	assert.NotNil(t, prop.AdditionalProperties)
	if typeName != "" {
		assert.EqualT(t, typeName, prop.AdditionalProperties.Schema.Type[0])
	}
	assert.Equal(t, goName, prop.Extensions["x-go-name"])
	assert.EqualT(t, format, prop.AdditionalProperties.Schema.Format)
}

func assertMapRef(t *testing.T, schema *oaispec.Schema, jsonName, goName, fragment string) {
	assertMapProperty(t, schema, "", jsonName, "", goName)
	psch := schema.Properties[jsonName].AdditionalProperties.Schema
	assert.EqualT(t, fragment, psch.Ref.String())
}

func TestBuilder_DiagnosticsOnInvalidNumeric(t *testing.T) {
	packagePattern := "./enhancements/diagnostics"
	packagePath := fixturesModule + "/enhancements/diagnostics"

	var collected []grammar.Diagnostic
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{packagePattern},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			collected = append(collected, d)
		},
	})
	require.NoError(t, err)

	decl, _ := ctx.FindDecl(packagePath, "BadMaximum")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "BadMaximum")]
	require.Contains(t, schema.Properties, "count")
	count := schema.Properties["count"]
	// The invalid `maximum: notanumber` is silently dropped — Maximum
	// stays nil on the property schema.
	assert.Nil(t, count.Maximum, "invalid maximum: should be dropped from spec")

	// Builder.Diagnostics() and the OnDiagnostic callback both surface
	// the parser's CodeInvalidNumber error.
	bd := prs.Diagnostics()
	require.NotEmpty(t, bd)
	require.NotEmpty(t, collected)

	foundCallback := false
	for _, d := range collected {
		if d.Code == grammar.CodeInvalidNumber {
			foundCallback = true
			break
		}
	}
	assert.True(t, foundCallback, "OnDiagnostic should fire with CodeInvalidNumber")

	foundBuilder := false
	for _, d := range bd {
		if d.Code == grammar.CodeInvalidNumber {
			foundBuilder = true
			break
		}
	}
	assert.True(t, foundBuilder, "Builder.Diagnostics should contain CodeInvalidNumber")
}

// TestBuilder_DiagnosticsOnAmbiguousEmbed exercises the
// embed-ambiguity diagnostic path. The fixture defines three
// shapes that all share a property JSON name across embeds:
//
//   - AmbiguousEmbed       — two sibling embeds at the same depth
//     promote the same JSON name under different Go field names;
//     the diagnostic must fire.
//   - DepthShadowingEmbed  — an inner embed at depth 2 is shadowed
//     by a Go field at depth 1; Go's depth rule already disambiguates
//     and the diagnostic must remain silent.
//   - ExplicitOverride     — a top-level field re-declares the
//     embedded JSON name; the embed-side override is happening at
//     depth 0 and the diagnostic must remain silent.
//
// The diagnostic carries CodeAmbiguousEmbed (SeverityWarning); the
// spec output remains last-write-wins regardless. Behaviour is not
// changed by this signal, only surfaced.
func TestBuilder_DiagnosticsOnAmbiguousEmbed(t *testing.T) {
	packagePattern := "./enhancements/diagnostics"
	packagePath := fixturesModule + "/enhancements/diagnostics"

	build := func(t *testing.T, name string) *Builder {
		t.Helper()
		ctx, err := scanner.NewScanCtx(&scanner.Options{
			Packages: []string{packagePattern},
			WorkDir:  scantest.FixturesDir(),
		})
		require.NoError(t, err)

		decl, _ := ctx.FindDecl(packagePath, name)
		require.NotNil(t, decl, "fixture decl %s not found", name)
		prs := NewBuilder(ctx, decl)
		models := make(map[string]oaispec.Schema)
		require.NoError(t, prs.Build(WithDefinitions(models)))
		return prs
	}

	hasAmbig := func(ds []grammar.Diagnostic) bool {
		for _, d := range ds {
			if d.Code == grammar.CodeAmbiguousEmbed {
				return true
			}
		}
		return false
	}

	t.Run("peer embeds at same depth fire the diagnostic", func(t *testing.T) {
		prs := build(t, "AmbiguousEmbed")
		ds := prs.Diagnostics()
		require.NotEmpty(t, ds)
		assert.True(t, hasAmbig(ds), "expected CodeAmbiguousEmbed in %+v", ds)
		// Verify severity and message shape.
		for _, d := range ds {
			if d.Code != grammar.CodeAmbiguousEmbed {
				continue
			}
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
			assert.Contains(t, d.Message, "shared")
			assert.Contains(t, d.Message, "Foo")
			assert.Contains(t, d.Message, "Bar")
		}
	})

	t.Run("depth-rule shadowing stays silent", func(t *testing.T) {
		prs := build(t, "DepthShadowingEmbed")
		assert.False(t, hasAmbig(prs.Diagnostics()),
			"depth shadowing must not be flagged as ambiguity")
	})

	t.Run("top-level explicit override stays silent", func(t *testing.T) {
		prs := build(t, "ExplicitOverride")
		assert.False(t, hasAmbig(prs.Diagnostics()),
			"top-level explicit override must not be flagged as ambiguity")
	})
}

// TestEmbeddedDescriptionAndTags verifies the allOf compound shape
// for $ref'd fields with field-level x-extensions and example. v1
// rode them as siblings of $ref (rejecting JSON Schema draft-4);
// the current builder produces the principled allOf compound where
// the description lives on the outer parent and the override
// decorations live on the override arm — see
// `internal/builders/schema/walker.go#applyToRefField` for the
// shape rules and the DescWithRef toggle's role.
func TestEmbeddedDescriptionAndTags(t *testing.T) {
	packagePattern := "./" + fixtureMinimal3125
	packagePath := fixturesModule + "/" + fixtureMinimal3125
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages: []string{packagePattern},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	decl, _ := ctx.FindDecl(packagePath, "Item")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "Item")]

	assert.Equal(t, []string{sampleValue1, sampleValue2}, schema.Required)
	require.Len(t, schema.Properties, 2)

	// Both Value1 and Value2 are typed *ValueStruct / ValueStruct
	// (named) → $ref. Field-level decorations move to the override
	// arm of an allOf compound; description rides the outer parent.

	// Vendor extensions ride the OUTER compound (alongside x-go-name)
	// so the field carries all its x-* metadata at one level.
	// Validations go on the override arm (AllOf[1]).

	require.MapContainsT(t, schema.Properties, sampleValue1)
	v1 := schema.Properties[sampleValue1]
	assert.EqualT(t, "Nullable value", v1.Description)
	assert.Equal(t, true, v1.Extensions["x-nullable"], "x-nullable should be on the outer compound, not inside AllOf")
	require.Len(t, v1.AllOf, 1, "value1 has an extension-only override → single-arm allOf")
	assert.Equal(t, "#/definitions/"+fixturesModule+"/"+fixtureMinimal3125+"/ValueStruct", v1.AllOf[0].Ref.String())
	assert.Empty(t, v1.Ref.String(), "outer schema must NOT carry the ref directly")

	require.MapContainsT(t, schema.Properties, sampleValue2)
	v2 := schema.Properties[sampleValue2]
	assert.EqualT(t, "Non-nullable value", v2.Description)
	assert.MapNotContainsT(t, v2.Extensions, "x-nullable")
	require.Len(t, v2.AllOf, 2, "value2 has an example override → two-arm allOf")
	assert.Equal(t, "#/definitions/"+fixturesModule+"/"+fixtureMinimal3125+"/ValueStruct", v2.AllOf[0].Ref.String())
	// The JSON-object example coerces structurally on the $ref override
	// arm, matching the direct-field path (quirk G3) — it was previously
	// carried as the raw string `{"value": 42}`.
	assert.Equal(t, map[string]any{"value": float64(42)}, v2.AllOf[1].Example)
}

// TestEmbeddedDescriptionAndTags_OptionVariants captures the
// (SkipExtensions, DescWithRef) option matrix on the bugs/3125
// fixture into separately-named goldens. Verifies that:
//
//   - Validation/extension overrides on a $ref'd field always wrap
//     in allOf (Value1's x-nullable, Value2's example are both
//     overrides). DescWithRef toggles description placement on the
//     allOf parent vs. dropped in the description-only-no-overrides
//     case (which doesn't apply here — both fields have overrides).
//   - SkipExtensions suppresses scanner-derived x-go-name /
//     x-go-package without affecting user-authored x-nullable or
//     the allOf shape.
//
// The four goldens produce a complete trace of (skipExt, descRef)
// permutations and serve as regression locks for the option
// semantics described in scanner.Options.
func TestEmbeddedDescriptionAndTags_OptionVariants(t *testing.T) {
	cases := []struct {
		name    string
		skipExt bool
		descRef bool
	}{
		{"default", false, false},
		{"DescWithRef", false, true},
		{"SkipExt", true, false},
		{"SkipExt+DescWithRef", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := scanner.NewScanCtx(&scanner.Options{
				Packages:       []string{"./" + fixtureMinimal3125},
				WorkDir:        scantest.FixturesDir(),
				SkipExtensions: tc.skipExt,
				DescWithRef:    tc.descRef,
			})
			require.NoError(t, err)
			decl, _ := ctx.FindDecl(fixturesModule+"/"+fixtureMinimal3125, "Item")
			require.NotNil(t, decl)
			prs := NewBuilder(ctx, decl)
			models := make(map[string]oaispec.Schema)
			require.NoError(t, prs.Build(WithDefinitions(models)))
			schema := models[scantest.ResolveTestKey(t, models, "Item")]

			require.MapContainsT(t, schema.Properties, sampleValue1)
			v1 := schema.Properties[sampleValue1]

			// Both fields have overrides → the allOf compound persists in
			// every option combination.
			require.NotEmpty(t, v1.AllOf, "overrides always wrap in allOf")
			assert.Empty(t, v1.Ref.String(), "outer schema must not carry the ref directly")

			// User-authored x-nullable survives regardless of SkipExtensions.
			assert.Equal(t, true, v1.Extensions["x-nullable"])

			if tc.skipExt {
				// SkipExtensions suppresses scanner-derived metadata.
				assert.MapNotContainsT(t, v1.Extensions, "x-go-name")
				assert.MapNotContainsT(t, v1.Extensions, "x-go-package")
			} else {
				assert.Equal(t, "Value1", v1.Extensions["x-go-name"])
			}

			// DescWithRef only governs the description-only-no-override case;
			// here both fields keep their descriptions on the outer compound.
			assert.EqualT(t, "Nullable value", v1.Description)
		})
	}
}

// TestEmbeddedDescriptionAndTags_SkipExtensions verifies that with
// SkipExtensions=true, the allOf compound on a $ref'd field is NOT
// polluted by the scanner-derived metadata (x-go-name / x-go-package
// / x-nullable inferred from pointer-ness). User-authored
// `Extensions: x-foo` blocks would still flow (they're explicit), but
// nothing else should land alongside the $ref.
//
// This is a regression guard: in v1, $ref'd fields had ps.Ref non-empty
// throughout, so the schema.go x-go-name / x-go-package guards
// (`if ps.Ref.String() == ""`) silently skipped. Post-S7, the allOf
// rewrite clears ps.Ref — those guards now fire. Without
// SkipExtensions=true, x-go-name lands on the outer compound (visible
// in the regular TestEmbeddedDescriptionAndTags). With
// SkipExtensions=true, the metadata extension writers respect the
// option and the outer compound stays clean.
func TestEmbeddedDescriptionAndTags_SkipExtensions(t *testing.T) {
	packagePattern := "./" + fixtureMinimal3125
	packagePath := fixturesModule + "/" + fixtureMinimal3125
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:       []string{packagePattern},
		WorkDir:        scantest.FixturesDir(),
		SkipExtensions: true,
	})
	require.NoError(t, err)
	decl, _ := ctx.FindDecl(packagePath, "Item")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "Item")]

	require.Len(t, schema.Properties, 2)

	// User-authored x-nullable should still be present (`Extensions:`
	// raw block in the source). Scanner-derived x-go-name, x-go-package
	// should be skipped.
	v1 := schema.Properties[sampleValue1]
	assert.MapNotContainsT(t, v1.Extensions, "x-go-name", "x-go-name should be skipped under SkipExtensions=true")
	assert.MapNotContainsT(t, v1.Extensions, "x-go-package", "x-go-package should be skipped under SkipExtensions=true")
	// Note: x-nullable on Value1 is user-authored, not scanner-derived;
	// it travels with the user's `Extensions:` block and SHOULD still
	// be present even under SkipExtensions=true.
	assert.Equal(t, true, v1.Extensions["x-nullable"], "user-authored x-nullable should survive SkipExtensions=true")

	v2 := schema.Properties[sampleValue2]
	assert.MapNotContainsT(t, v2.Extensions, "x-go-name")
	assert.MapNotContainsT(t, v2.Extensions, "x-go-package")
	assert.MapNotContainsT(t, v2.Extensions, "x-nullable", "value2 has no x-nullable in source")
}

// TestEmbeddedDescriptionAndTags_SkipAllOfCompounding is the A/B
// witness for the SkipAllOfCompounding option on the bugs/3125
// fixture. Both Value1 (*ValueStruct, user-authored x-nullable +
// description) and Value2 (ValueStruct, example + description) are
// $ref'd fields whose siblings normally wrap into an allOf compound
// (see TestEmbeddedDescriptionAndTags).
//
// With SkipAllOfCompounding=true:
//
//   - each field emits a BARE {$ref} — no AllOf wrapper, no
//     description, no override extension/example sibling;
//   - `required` (a parent-side concern) is PRESERVED on the enclosing
//     object — it is not a $ref sibling;
//   - every dropped sibling raises one CodeDroppedRefSibling diagnostic
//     (x-nullable, example, and one per dropped description) so the
//     loss is never silent.
func TestEmbeddedDescriptionAndTags_SkipAllOfCompounding(t *testing.T) {
	packagePath := fixturesModule + "/" + fixtureMinimal3125

	var diags []grammar.Diagnostic
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:             []string{"./" + fixtureMinimal3125},
		WorkDir:              scantest.FixturesDir(),
		SkipAllOfCompounding: true,
		OnDiagnostic:         func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	decl, _ := ctx.FindDecl(packagePath, "Item")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "Item")]

	require.Len(t, schema.Properties, 2)

	// required is parent-side: preserved even with compounding disabled.
	assert.ElementsMatch(t, []string{sampleValue1, sampleValue2}, schema.Required)

	refFrag := "#/definitions/" + fixturesModule + "/" + fixtureMinimal3125 + "/ValueStruct"

	// Value1: bare $ref — no allOf, no description, no x-nullable sibling.
	v1 := schema.Properties[sampleValue1]
	assert.Equal(t, refFrag, v1.Ref.String(), "value1 must be a bare $ref")
	assert.Empty(t, v1.AllOf, "no allOf compound when compounding is disabled")
	assert.Empty(t, v1.Description, "description dropped on a bare $ref")
	assert.MapNotContainsT(t, v1.Extensions, "x-nullable", "override extension dropped on a bare $ref")

	// Value2: bare $ref — no allOf, no description, no example sibling.
	v2 := schema.Properties[sampleValue2]
	assert.Equal(t, refFrag, v2.Ref.String(), "value2 must be a bare $ref")
	assert.Empty(t, v2.AllOf, "no allOf compound when compounding is disabled")
	assert.Empty(t, v2.Description, "description dropped on a bare $ref")
	assert.Nil(t, v2.Example, "override example dropped on a bare $ref")

	// Every dropped sibling is reported. Value1 → x-nullable + description;
	// Value2 → example + description. All carry CodeDroppedRefSibling.
	var keywords, descDrops int
	for _, d := range diags {
		if d.Code != grammar.CodeDroppedRefSibling {
			continue
		}
		switch {
		case strings.Contains(d.Message, "description dropped"):
			descDrops++
		default:
			keywords++
		}
	}
	assert.Equal(t, 2, descDrops, "one description-drop diagnostic per $ref'd field")
	assert.Equal(t, 2, keywords, "x-nullable and example each raise a drop diagnostic")

	msgs := make([]string, 0, len(diags))
	for _, d := range diags {
		msgs = append(msgs, d.Message)
	}
	assert.True(t, sliceContainsSubstr(msgs, "x-nullable"), "x-nullable drop reported: %v", msgs)
	assert.True(t, sliceContainsSubstr(msgs, "example"), "example drop reported: %v", msgs)
}

func sliceContainsSubstr(ss []string, sub string) bool {
	for _, s := range ss {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// TestEmbeddedDescriptionAndTags_EmitRefSiblings exercises the
// EmitRefSiblings lenient mode (no SkipAllOfCompounding) on bugs/3125.
//
//   - Value1 (*ValueStruct, user x-nullable + description): no
//     validation forces a compound, so description and x-nullable ride
//     as DIRECT $ref siblings — bare {$ref, description, x-nullable},
//     no allOf.
//   - Value2 (ValueStruct, example + description): `example` is a
//     validation-class override, which still forces an allOf compound;
//     the description rides the outer compound and the example lands on
//     the override arm. EmitRefSiblings does not change the
//     forced-compound case.
func TestEmbeddedDescriptionAndTags_EmitRefSiblings(t *testing.T) {
	packagePath := fixturesModule + "/" + fixtureMinimal3125

	var diags []grammar.Diagnostic
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:        []string{"./" + fixtureMinimal3125},
		WorkDir:         scantest.FixturesDir(),
		EmitRefSiblings: true,
		OnDiagnostic:    func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	decl, _ := ctx.FindDecl(packagePath, "Item")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "Item")]

	require.Len(t, schema.Properties, 2)
	assert.ElementsMatch(t, []string{sampleValue1, sampleValue2}, schema.Required)

	refFrag := "#/definitions/" + fixturesModule + "/" + fixtureMinimal3125 + "/ValueStruct"

	// Value1: direct siblings, no allOf.
	v1 := schema.Properties[sampleValue1]
	assert.Equal(t, refFrag, v1.Ref.String(), "value1 keeps a bare $ref")
	assert.Empty(t, v1.AllOf, "no allOf wrap for sibling-eligible decoration")
	assert.EqualT(t, "Nullable value", v1.Description, "description rides as a $ref sibling")
	assert.Equal(t, true, v1.Extensions["x-nullable"], "extension rides as a $ref sibling")

	// Value2: a validation (example) still forces the compound.
	v2 := schema.Properties[sampleValue2]
	assert.Empty(t, v2.Ref.String(), "value2 outer carries no ref — it is a compound")
	require.Len(t, v2.AllOf, 2, "example (a validation) forces a two-arm allOf")
	assert.Equal(t, refFrag, v2.AllOf[0].Ref.String())
	assert.EqualT(t, "Non-nullable value", v2.Description, "description rides the outer compound")
	assert.Equal(t, map[string]any{"value": float64(42)}, v2.AllOf[1].Example)

	// Nothing was dropped → no drop diagnostics.
	for _, d := range diags {
		assert.NotEqual(t, grammar.CodeDroppedRefSibling, d.Code, "unexpected drop: %s", d.Message)
	}
}

// TestEmbeddedDescriptionAndTags_EmitRefSiblings_Skip covers the
// EmitRefSiblings + SkipAllOfCompounding combination on bugs/3125
// (the "both on" quadrant). No allOf compound is ever produced:
//
//   - Value1: description + x-nullable survive as direct $ref siblings.
//   - Value2: description survives as a sibling, but `example` (a
//     validation, which can only ride a compound) is dropped with a
//     diagnostic.
func TestEmbeddedDescriptionAndTags_EmitRefSiblings_Skip(t *testing.T) {
	packagePath := fixturesModule + "/" + fixtureMinimal3125

	var diags []grammar.Diagnostic
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:             []string{"./" + fixtureMinimal3125},
		WorkDir:              scantest.FixturesDir(),
		EmitRefSiblings:      true,
		SkipAllOfCompounding: true,
		OnDiagnostic:         func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	decl, _ := ctx.FindDecl(packagePath, "Item")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "Item")]

	require.Len(t, schema.Properties, 2)
	assert.ElementsMatch(t, []string{sampleValue1, sampleValue2}, schema.Required)

	refFrag := "#/definitions/" + fixturesModule + "/" + fixtureMinimal3125 + "/ValueStruct"

	// Value1: siblings survive (no compound ever needed).
	v1 := schema.Properties[sampleValue1]
	assert.Equal(t, refFrag, v1.Ref.String())
	assert.Empty(t, v1.AllOf)
	assert.EqualT(t, "Nullable value", v1.Description)
	assert.Equal(t, true, v1.Extensions["x-nullable"])

	// Value2: description survives as a sibling; example (validation) dropped.
	v2 := schema.Properties[sampleValue2]
	assert.Equal(t, refFrag, v2.Ref.String(), "value2 stays a bare $ref")
	assert.Empty(t, v2.AllOf, "no compound produced under SkipAllOfCompounding")
	assert.EqualT(t, "Non-nullable value", v2.Description, "description survives as a sibling")
	assert.Nil(t, v2.Example, "example dropped — a validation cannot ride a bare $ref")

	var drops int
	for _, d := range diags {
		if d.Code == grammar.CodeDroppedRefSibling {
			drops++
		}
	}
	assert.Equal(t, 1, drops, "only the example (a validation) is dropped")
}

// TestParamsShape_DescWithRef_BothModes covers the description-only
// $ref'd field case where the user toggles DescWithRef:
//
//   - DescWithRef=false (default): the description is dropped and the
//     field emits as a bare {$ref: ...}.
//   - DescWithRef=true: the description rides a single-arm allOf
//     compound — {description: ..., allOf: [{$ref}]}.
//
// Fixture: classification operations corpus' `pet` field of
// `items[]` in NoModel carries only a description plus a $ref to the
// pet model — no validations, no user-authored extensions.
//
// When the field carries validation or extension overrides, the
// allOf compound is mandatory regardless of DescWithRef — covered by
// TestEmbeddedDescriptionAndTags / TestEmbeddedDescriptionAndTags_SkipExtensions.
func TestParamsShape_DescWithRef_BothModes(t *testing.T) {
	getPetField := func(t *testing.T, descWithRef bool) oaispec.Schema {
		t.Helper()
		ctx, err := scanner.NewScanCtx(&scanner.Options{
			Packages: []string{
				"./goparsing/classification",
				"./goparsing/classification/models",
				"./goparsing/classification/operations",
			},
			WorkDir:        scantest.FixturesDir(),
			SkipExtensions: true,
			DescWithRef:    descWithRef,
		})
		require.NoError(t, err)
		decl, ok := ctx.FindDecl(fixturesModule+"/goparsing/classification/models", "NoModel")
		require.True(t, ok)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		models := make(map[string]oaispec.Schema)
		require.NoError(t, prs.Build(WithDefinitions(models)))
		noModel := models[scantest.ResolveTestKey(t, models, "NoModel")]
		require.Contains(t, noModel.Properties, "items")
		itemsProp := noModel.Properties["items"]
		require.NotNil(t, itemsProp.Items)
		require.NotNil(t, itemsProp.Items.Schema)
		itemSchema := itemsProp.Items.Schema
		require.Contains(t, itemSchema.Properties, "pet")
		return itemSchema.Properties["pet"]
	}

	t.Run("DescWithRef=false → bare $ref", func(t *testing.T) {
		pet := getPetField(t, false)
		assert.Equal(t, "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/pet", pet.Ref.String())
		assert.Empty(t, pet.AllOf, "no allOf compound expected")
		assert.Empty(t, pet.Description, "description dropped under DescWithRef=false")
		assert.MapNotContainsT(t, pet.Extensions, "x-go-name")
	})

	t.Run("DescWithRef=true → single-arm allOf with description", func(t *testing.T) {
		pet := getPetField(t, true)
		assert.Empty(t, pet.Ref.String(), "outer schema must NOT carry the ref directly")
		require.Len(t, pet.AllOf, 1, "single-arm allOf for description-only override")
		assert.Equal(t, "#/definitions/"+fixturesModule+"/goparsing/classification/transitive/mods/pet", pet.AllOf[0].Ref.String())
		assert.Contains(t, pet.Description, "The Pet to add")
		assert.MapNotContainsT(t, pet.Extensions, "x-go-name")
	})
}

// TestIssue2540 verifies the JSON Schema draft-4 allOf compound shape
// for a $ref'd field (`Author Author`) carrying its own field-level
// `example:`. The example must travel on the override arm of the
// allOf compound, never as a sibling of $ref. The DescWithRef toggle
// does not change this case — when validations (here, `example`)
// are present, the allOf wrap is mandatory regardless of the flag.
func TestIssue2540(t *testing.T) {
	// Sub-builder unit tests run without the spec reduce stage, so the
	// definitions key and the $ref stay fully-qualified.
	const expectedJSON = `{
		"github.com/go-openapi/codescan/fixtures/bugs/2540/foo/Book": {
      "description": "At this moment, a book is only described by its publishing date\nand author.",
      "type": "object",
      "title": "Book holds all relevant information about a book.",
			"example": "{ \"Published\": 2026, \"Author\": \"Fred\" }",
      "default": "{ \"Published\": 1900, \"Author\": \"Unknown\" }",
      "properties": {
        "Author": {
          "allOf": [
            {"$ref": "#/definitions/github.com/go-openapi/codescan/fixtures/bugs/2540/foo/Author"},
            {"example": {"Name": "Tolkien"}}
          ]
        },
        "Published": {
          "type": "integer",
          "format": "int64",
          "minimum": 0,
          "example": 2021
        }
      }
    }
  }`
	packagePattern := "./bugs/2540/foo"
	packagePath := fixturesModule + "/bugs/2540/foo"
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:       []string{packagePattern},
		WorkDir:        scantest.FixturesDir(),
		SkipExtensions: true,
	})
	require.NoError(t, err)

	decl, _ := ctx.FindDecl(packagePath, "Book")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	b, err := json.Marshal(models)
	require.NoError(t, err)
	assert.JSONEqT(t, expectedJSON, string(b))
}
