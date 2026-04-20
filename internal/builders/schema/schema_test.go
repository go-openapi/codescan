// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"testing"

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
)

func TestBuilder_Struct_Tag(t *testing.T) {
	ctx := scantest.LoadPetstorePkgsCtx(t, false)

	var td *scanner.EntityDecl
	t.Run("should find a Tag model", func(t *testing.T) {
		for k, v := range ctx.Models() {
			if k.Name != "Tag" {
				continue
			}
			td = v
			break
		}
		require.NotNil(t, td)
	})

	prs := &Builder{
		ctx:  ctx,
		decl: td,
	}
	result := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(result))

	scantest.CompareOrDumpJSON(t, result, "petstore_schema_Tag.json")
}

func TestBuilder_Struct_Pet(t *testing.T) {
	// Debug = true
	// defer func() { Debug = false }()

	ctx := scantest.LoadPetstorePkgsCtx(t, false)
	var td *scanner.EntityDecl
	for k, v := range ctx.Models() {
		if k.Name != "Pet" {
			continue
		}
		td = v
		break
	}
	require.NotNil(t, td)

	prs := &Builder{
		ctx:  ctx,
		decl: td,
	}
	result := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(result))

	scantest.CompareOrDumpJSON(t, result, "petstore_schema_Pet.json")
}

func TestBuilder_Struct_Order(t *testing.T) {
	// Debug = true
	// defer func() { Debug = false }()

	ctx := scantest.LoadPetstorePkgsCtx(t, false)
	var td *scanner.EntityDecl
	for k, v := range ctx.Models() {
		if k.Name != "Order" {
			continue
		}
		td = v
		break
	}
	require.NotNil(t, td)

	prs := &Builder{
		ctx:  ctx,
		decl: td,
	}
	result := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(result))

	scantest.CompareOrDumpJSON(t, result, "petstore_schema_Order.json")
}

func TestBuilder(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "NoModel")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["NoModel"]

	assert.Equal(t, oaispec.StringOrArray([]string{"object"}), schema.Type)
	assert.EqualT(t, "NoModel is a struct without an annotation.", schema.Title)
	assert.EqualT(t, "NoModel exists in a package\nbut is not annotated with the swagger model annotations\nso it should now show up in a test.", schema.Description)
	assert.Len(t, schema.Required, 3)
	assert.Len(t, schema.Properties, 12)

	scantest.AssertProperty(t, &schema, "integer", "id", "int64", "ID")
	prop, ok := schema.Properties["id"]
	assert.EqualT(t, "ID of this no model instance.\nids in this application start at 11 and are smaller than 1000", prop.Description)
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
			"value1",
			"value2",
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
	assert.EqualT(t, "ID of this no model instance.\nids in this application start at 11 and are smaller than 1000", iprop.Description)
	require.NotNil(t, iprop.Maximum)
	assert.InDeltaT(t, 1000.00, *iprop.Maximum, epsilon)
	assert.TrueT(t, iprop.ExclusiveMaximum, "'id' should have had an exclusive maximum")
	require.NotNil(t, iprop.Minimum)
	assert.InDeltaT(t, 10.00, *iprop.Minimum, epsilon)
	assert.TrueT(t, iprop.ExclusiveMinimum, "'id' should have had an exclusive minimum")
	assert.Equal(t, 11, iprop.Default, "ID default value is incorrect")

	scantest.AssertRef(t, itprop, "pet", "Pet", "#/definitions/pet")
	iprop, ok = itprop.Properties["pet"]
	assert.TrueT(t, ok)
	if itprop.Ref.String() != "" {
		assert.EqualT(t, "The Pet to add to this NoModel items bucket.\nPets can appear more than once in the bucket", iprop.Description)
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
	require.NoError(t, (&Builder{decl: decl2, ctx: ctx}).Build(models))
	msch, ok := models["order"]
	pn := fixturesModule + "/goparsing/classification/models"
	assert.TrueT(t, ok)
	assert.Equal(t, pn, msch.Extensions["x-go-package"])
	assert.Equal(t, "StoreOrder", msch.Extensions["x-go-name"])

	scantest.CompareOrDumpJSON(t, models, "classification_schema_NoModel.json")
}

func TestBuilder_AddExtensions(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	models := make(map[string]oaispec.Schema)
	decl := getClassificationModel(ctx, "StoreOrder")
	require.NotNil(t, decl)
	require.NoError(t, (&Builder{decl: decl, ctx: ctx}).Build(models))

	msch, ok := models["order"]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["TextMarshalModel"]
	scantest.AssertProperty(t, &schema, "string", "id", "uuid", "ID")
	scantest.AssertArrayProperty(t, &schema, "string", "ids", "uuid", "IDs")
	scantest.AssertProperty(t, &schema, "string", "struct", "", "Struct")
	scantest.AssertProperty(t, &schema, "string", "map", "", "Map")
	assertMapProperty(t, &schema, "string", "mapUUID", "uuid", "MapUUID")
	scantest.AssertRef(t, &schema, "url", "URL", "#/definitions/URL")
	scantest.AssertProperty(t, &schema, "string", "time", "date-time", "Time")
	scantest.AssertProperty(t, &schema, "string", "structStrfmt", "date-time", "StructStrfmt")
	scantest.AssertProperty(t, &schema, "string", "structStrfmtPtr", "date-time", "StructStrfmtPtr")
	scantest.AssertProperty(t, &schema, "string", "customUrl", "url", "CustomURL")
}

func TestEmbeddedTypes(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "ComplexerOne")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["ComplexerOne"]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["PrimateModel"]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["FormattedModel"]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	sch := models["jsonString"]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	sch := models["jsonPtrString"]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	sch := models["ignoredFields"]
	scantest.AssertProperty(t, &sch, "string", "someIncludedField", "", "SomeIncludedField")
	scantest.AssertProperty(t, &sch, "string", "someErroneouslyIncludedField", "", "SomeErroneouslyIncludedField")
	assert.Len(t, sch.Properties, 2)
}

func TestParseStructFields(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "SimpleComplexModel")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["SimpleComplexModel"]
	scantest.AssertProperty(t, &schema, "object", "emb", "", "Emb")
	eSchema := schema.Properties["emb"]
	scantest.AssertProperty(t, &eSchema, "integer", "cid", "int64", "CID")
	scantest.AssertProperty(t, &eSchema, "string", "baz", "", "Baz")

	scantest.AssertRef(t, &schema, "top", "Top", "#/definitions/Something")
	scantest.AssertRef(t, &schema, "notSel", "NotSel", "#/definitions/NotSelected")
}

func TestParsePointerFields(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "Pointdexter")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["Pointdexter"]

	scantest.AssertProperty(t, &schema, "integer", "id", "int64", "ID")
	scantest.AssertProperty(t, &schema, "string", "name", "", "Name")
	scantest.AssertProperty(t, &schema, "object", "emb", "", "Emb")
	scantest.AssertProperty(t, &schema, "string", "t", "uuid5", "T")
	eSchema := schema.Properties["emb"]
	scantest.AssertProperty(t, &eSchema, "integer", "cid", "int64", "CID")
	scantest.AssertProperty(t, &eSchema, "string", "baz", "", "Baz")

	scantest.AssertRef(t, &schema, "top", "Top", "#/definitions/Something")
	scantest.AssertRef(t, &schema, "notSel", "NotSel", "#/definitions/NotSelected")
}

func TestEmbeddedStarExpr(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "EmbeddedStarExpr")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["EmbeddedStarExpr"]

	scantest.AssertProperty(t, &schema, "integer", "embeddedMember", "int64", "EmbeddedMember")
	scantest.AssertProperty(t, &schema, "integer", "notEmbedded", "int64", "NotEmbedded")
}

func TestArrayOfPointers(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "Cars")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["cars"]
	scantest.AssertProperty(t, &schema, "array", "cars", "", "Cars")
}

func TestOverridingOneIgnore(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "OverridingOneIgnore")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["OverridingOneIgnore"]

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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models[modelName]

	ca.assertProperty(t, &schema, "integer", "ids", "int64", "IDs")
	ca.assertProperty(t, &schema, "string", "names", "", "Names")
	ca.assertProperty(t, &schema, "string", "uuids", "uuid", "UUIDs")
	ca.assertProperty(t, &schema, "object", "embs", "", "Embs")
	eSchema := ca.nestedSchema(schema.Properties["embs"])
	ca.assertProperty(t, eSchema, "integer", "cid", "int64", "CID")
	ca.assertProperty(t, eSchema, "string", "baz", "", "Baz")

	ca.assertRef(t, &schema, "tops", "Tops", "#/definitions/Something")
	ca.assertRef(t, &schema, "notSels", "NotSels", "#/definitions/NotSelected")

	ca.assertProperty(t, &schema, "integer", "ptrIds", "int64", "PtrIDs")
	ca.assertProperty(t, &schema, "string", "ptrNames", "", "PtrNames")
	ca.assertProperty(t, &schema, "string", "ptrUuids", "uuid", "PtrUUIDs")
	ca.assertProperty(t, &schema, "object", "ptrEmbs", "", "PtrEmbs")
	eSchema = ca.nestedSchema(schema.Properties["ptrEmbs"])
	ca.assertProperty(t, eSchema, "integer", "ptrCid", "int64", "PtrCID")
	ca.assertProperty(t, eSchema, "string", "ptrBaz", "", "PtrBaz")

	ca.assertRef(t, &schema, "ptrTops", "PtrTops", "#/definitions/Something")
	ca.assertRef(t, &schema, "ptrNotSels", "PtrNotSels", "#/definitions/NotSelected")
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["Interfaced"]
	scantest.AssertProperty(t, &schema, "", "custom_data", "", "CustomData")
}

func TestAliasedTypes(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "OtherTypes")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["OtherTypes"]
	scantest.AssertRef(t, &schema, "named", "Named", "#/definitions/SomeStringType")
	scantest.AssertRef(t, &schema, "numbered", "Numbered", "#/definitions/SomeIntType")
	scantest.AssertProperty(t, &schema, "string", "dated", "date-time", "Dated")
	scantest.AssertRef(t, &schema, "timed", "Timed", "#/definitions/SomeTimedType")
	scantest.AssertRef(t, &schema, "petted", "Petted", "#/definitions/SomePettedType")
	scantest.AssertRef(t, &schema, "somethinged", "Somethinged", "#/definitions/SomethingType")
	scantest.AssertRef(t, &schema, "strMap", "StrMap", "#/definitions/SomeStringMap")
	scantest.AssertRef(t, &schema, "strArrMap", "StrArrMap", "#/definitions/SomeArrayStringMap")

	scantest.AssertRef(t, &schema, "manyNamed", "ManyNamed", "#/definitions/SomeStringsType")
	scantest.AssertRef(t, &schema, "manyNumbered", "ManyNumbered", "#/definitions/SomeIntsType")
	scantest.AssertArrayProperty(t, &schema, "string", "manyDated", "date-time", "ManyDated")
	scantest.AssertRef(t, &schema, "manyTimed", "ManyTimed", "#/definitions/SomeTimedsType")
	scantest.AssertRef(t, &schema, "manyPetted", "ManyPetted", "#/definitions/SomePettedsType")
	scantest.AssertRef(t, &schema, "manySomethinged", "ManySomethinged", "#/definitions/SomethingsType")

	assertArrayRef(t, &schema, "nameds", "Nameds", "#/definitions/SomeStringType")
	assertArrayRef(t, &schema, "numbereds", "Numbereds", "#/definitions/SomeIntType")
	scantest.AssertArrayProperty(t, &schema, "string", "dateds", "date-time", "Dateds")
	assertArrayRef(t, &schema, "timeds", "Timeds", "#/definitions/SomeTimedType")
	assertArrayRef(t, &schema, "petteds", "Petteds", "#/definitions/SomePettedType")
	assertArrayRef(t, &schema, "somethingeds", "Somethingeds", "#/definitions/SomethingType")

	scantest.AssertRef(t, &schema, "modsNamed", "ModsNamed", "#/definitions/modsSomeStringType")
	scantest.AssertRef(t, &schema, "modsNumbered", "ModsNumbered", "#/definitions/modsSomeIntType")
	scantest.AssertProperty(t, &schema, "string", "modsDated", "date-time", "ModsDated")
	scantest.AssertRef(t, &schema, "modsTimed", "ModsTimed", "#/definitions/modsSomeTimedType")
	scantest.AssertRef(t, &schema, "modsPetted", "ModsPetted", "#/definitions/modsSomePettedType")

	assertArrayRef(t, &schema, "modsNameds", "ModsNameds", "#/definitions/modsSomeStringType")
	assertArrayRef(t, &schema, "modsNumbereds", "ModsNumbereds", "#/definitions/modsSomeIntType")
	scantest.AssertArrayProperty(t, &schema, "string", "modsDateds", "date-time", "ModsDateds")
	assertArrayRef(t, &schema, "modsTimeds", "ModsTimeds", "#/definitions/modsSomeTimedType")
	assertArrayRef(t, &schema, "modsPetteds", "ModsPetteds", "#/definitions/modsSomePettedType")

	scantest.AssertRef(t, &schema, "manyModsNamed", "ManyModsNamed", "#/definitions/modsSomeStringsType")
	scantest.AssertRef(t, &schema, "manyModsNumbered", "ManyModsNumbered", "#/definitions/modsSomeIntsType")
	scantest.AssertArrayProperty(t, &schema, "string", "manyModsDated", "date-time", "ManyModsDated")
	scantest.AssertRef(t, &schema, "manyModsTimed", "ManyModsTimed", "#/definitions/modsSomeTimedsType")
	scantest.AssertRef(t, &schema, "manyModsPetted", "ManyModsPetted", "#/definitions/modsSomePettedsType")
	scantest.AssertRef(t, &schema, "manyModsPettedPtr", "ManyModsPettedPtr", "#/definitions/modsSomePettedsPtrType")

	scantest.AssertProperty(t, &schema, "string", "namedAlias", "", "NamedAlias")
	scantest.AssertProperty(t, &schema, "integer", "numberedAlias", "int64", "NumberedAlias")
	scantest.AssertArrayProperty(t, &schema, "string", "namedsAlias", "", "NamedsAlias")
	scantest.AssertArrayProperty(t, &schema, "integer", "numberedsAlias", "int64", "NumberedsAlias")
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

		prs := &Builder{
			decl: decl,
			ctx:  ctx,
		}
		require.NoError(t, prs.Build(defs))
	}

	for k := range defs {
		for i, b := range names {
			if b == k {
				// remove the entry from the collection
				names = append(names[:i], names[i+1:]...)
			}
		}
	}
	if assert.Empty(t, names) {
		// single value types
		assertDefinition(t, defs, "SomeStringType", "string", "")
		assertDefinition(t, defs, "SomeIntType", "integer", "int64")
		assertDefinition(t, defs, "SomeTimeType", "string", "date-time")
		assertDefinition(t, defs, "SomeTimedType", "string", "date-time")
		assertRefDefinition(t, defs, "SomePettedType", "#/definitions/pet", "")
		assertRefDefinition(t, defs, "SomethingType", "#/definitions/Something", "")

		// slice types
		assertArrayDefinition(t, defs, "SomeStringsType", "string", "", "")
		assertArrayDefinition(t, defs, "SomeIntsType", "integer", "int64", "")
		assertArrayDefinition(t, defs, "SomeTimesType", "string", "date-time", "")
		assertArrayDefinition(t, defs, "SomeTimedsType", "string", "date-time", "")
		assertArrayWithRefDefinition(t, defs, "SomePettedsType", "#/definitions/pet", "")
		assertArrayWithRefDefinition(t, defs, "SomethingsType", "#/definitions/Something", "")

		// map types
		assertMapDefinition(t, defs, "SomeObject", "object", "", "")
		assertMapDefinition(t, defs, "SomeStringMap", "string", "", "")
		assertMapDefinition(t, defs, "SomeIntMap", "integer", "int64", "")
		assertMapDefinition(t, defs, "SomeTimeMap", "string", "date-time", "")
		assertMapDefinition(t, defs, "SomeTimedMap", "string", "date-time", "")
		assertMapWithRefDefinition(t, defs, "SomePettedMap", "#/definitions/pet", "")
		assertMapWithRefDefinition(t, defs, "SomeSomethingMap", "#/definitions/Something", "")
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
				builder := &Builder{
					ctx:  ctx,
					decl: decl,
				}

				t.Run("should build model for Customer", func(t *testing.T) {
					models := make(map[string]oaispec.Schema)
					require.NoError(t, builder.Build(models))

					assertRefDefinition(t, models, "Customer", "#/definitions/User", "")
				})

				t.Run("should have discovered models for User and Customer", func(t *testing.T) {
					require.Len(t, builder.postDecls, 2)
					foundUserIndex := -1
					foundCustomerIndex := -1

					for i, discoveredDecl := range builder.postDecls {
						switch discoveredDecl.Obj().Name() {
						case "User":
							foundUserIndex = i
						case "Customer":
							foundCustomerIndex = i
						}
					}
					require.GreaterOrEqualT(t, foundUserIndex, 0)
					require.GreaterOrEqualT(t, foundCustomerIndex, 0)

					userBuilder := &Builder{
						ctx:  ctx,
						decl: builder.postDecls[foundUserIndex],
					}

					t.Run("should build model for User", func(t *testing.T) {
						models := make(map[string]oaispec.Schema)
						require.NoError(t, userBuilder.Build(models))

						require.MapContainsT(t, models, "User")

						user := models["User"]
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
				builder := &Builder{
					ctx:  ctx,
					decl: decl,
				}

				t.Run("should build model for Customer", func(t *testing.T) {
					models := make(map[string]oaispec.Schema)
					require.NoError(t, builder.Build(models))

					require.MapContainsT(t, models, "Customer")
					customer := models["Customer"]
					require.MapNotContainsT(t, models, "User")

					assert.TrueT(t, customer.Type.Contains("object"))

					customerProperties := customer.Properties
					assert.MapContainsT(t, customerProperties, "name")
					assert.NotEmpty(t, customer.Title)
				})

				t.Run("should have discovered only Customer", func(t *testing.T) {
					require.Len(t, builder.postDecls, 1)
					discovered := builder.postDecls[0]
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
	prs := &Builder{
		ctx:  ctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["AllOfModel"]

	require.Len(t, schema.AllOf, 3)
	asch := schema.AllOf[0]
	scantest.AssertProperty(t, &asch, "integer", "age", "int32", "Age")
	scantest.AssertProperty(t, &asch, "integer", "id", "int64", "ID")
	scantest.AssertProperty(t, &asch, "string", "name", "", "Name")

	asch = schema.AllOf[1]
	assert.EqualT(t, "#/definitions/withNotes", asch.Ref.String())

	asch = schema.AllOf[2]
	scantest.AssertProperty(t, &asch, "string", "createdAt", "date-time", "CreatedAt")
	scantest.AssertProperty(t, &asch, "integer", "did", "int64", "DID")
	scantest.AssertProperty(t, &asch, "string", "cat", "", "Cat")

	scantest.CompareOrDumpJSON(t, models, "classification_schema_AllOfModel.json")
}

func TestPointersAreNullableByDefaultWhenSetXNullableForPointersIsSet(t *testing.T) {
	allModels := make(map[string]oaispec.Schema)
	assertModel := func(ctx *scanner.ScanCtx, packagePath, modelName string) {
		decl, _ := ctx.FindDecl(packagePath, modelName)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(allModels))

		schema := allModels[modelName]
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

	scantest.CompareOrDumpJSON(t, allModels, "enhancements_pointers_xnullable.json")
}

// valueKeys returns the five property keys expected for the fixtures
// Item (struct, Go names verbatim) and ItemInterface (interface methods,
// camelCased per Q9).
func valueKeys(modelName string) (string, string, string, string, string) {
	if modelName == "ItemInterface" {
		return "value1", "value2", "value3", "value4", "value5"
	}
	return "Value1", "Value2", "Value3", "Value4", "Value5"
}

func TestPointersAreNotNullableByDefaultWhenSetXNullableForPointersIsNotSet(t *testing.T) {
	allModels := make(map[string]oaispec.Schema)
	assertModel := func(ctx *scanner.ScanCtx, packagePath, modelName string) {
		decl, _ := ctx.FindDecl(packagePath, modelName)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(allModels))

		schema := allModels[modelName]
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

	scantest.CompareOrDumpJSON(t, allModels, "enhancements_pointers_no_xnullable.json")
}

func TestSwaggerTypeNamed(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	decl := getClassificationModel(ctx, "NamedWithType")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["namedWithType"]

	scantest.AssertProperty(t, &schema, "object", "some_map", "", "SomeMap")

	scantest.CompareOrDumpJSON(t, models, "classification_schema_NamedWithType.json")
}

func TestSwaggerTypeNamedWithGenerics(t *testing.T) {
	tests := map[string]func(t *testing.T, models map[string]oaispec.Schema){
		"NamedStringResults": func(t *testing.T, models map[string]oaispec.Schema) {
			schema := models["namedStringResults"]
			scantest.AssertArrayProperty(t, &schema, "string", "matches", "", "Matches")
		},
		"NamedStoreOrderResults": func(t *testing.T, models map[string]oaispec.Schema) {
			schema := models["namedStoreOrderResults"]
			assertArrayRef(t, &schema, "matches", "Matches", "#/definitions/order")
		},
		"NamedStringSlice": func(t *testing.T, models map[string]oaispec.Schema) {
			assertArrayDefinition(t, models, "namedStringSlice", "string", "", "NamedStringSlice")
		},
		"NamedStoreOrderSlice": func(t *testing.T, models map[string]oaispec.Schema) {
			assertArrayWithRefDefinition(t, models, "namedStoreOrderSlice", "#/definitions/order", "NamedStoreOrderSlice")
		},
		"NamedStringMap": func(t *testing.T, models map[string]oaispec.Schema) {
			assertMapDefinition(t, models, "namedStringMap", "string", "", "NamedStringMap")
		},
		"NamedStoreOrderMap": func(t *testing.T, models map[string]oaispec.Schema) {
			assertMapWithRefDefinition(t, models, "namedStoreOrderMap", "#/definitions/order", "NamedStoreOrderMap")
		},
		"NamedMapOfStoreOrderSlices": func(t *testing.T, models map[string]oaispec.Schema) {
			assertMapDefinition(t, models, "namedMapOfStoreOrderSlices", "array", "", "NamedMapOfStoreOrderSlices")
			arraySchema := models["namedMapOfStoreOrderSlices"].AdditionalProperties.Schema
			assertArrayWithRefDefinition(t, map[string]oaispec.Schema{
				"array": *arraySchema,
			}, "array", "#/definitions/order", "")
		},
	}

	for testName, testFunc := range tests {
		t.Run(testName, func(t *testing.T) {
			ctx := scantest.LoadClassificationPkgsCtx(t)
			decl := getClassificationModel(ctx, testName)
			require.NotNil(t, decl)
			prs := NewBuilder(ctx, decl)
			models := make(map[string]oaispec.Schema)
			require.NoError(t, prs.Build(models))
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
	require.NoError(t, prs.Build(models))
	schema := models["NullString"]

	assert.TrueT(t, schema.Type.Contains("string"))

	scantest.CompareOrDumpJSON(t, models, "classification_schema_NullString.json")
}

func TestStructDiscriminators(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)

	models := make(map[string]oaispec.Schema)
	for _, tn := range []string{"BaseStruct", "Giraffe", "Gazelle"} {
		decl := getClassificationModel(ctx, tn)
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(models))
	}

	schema := models["animal"]

	assert.Equal(t, "BaseStruct", schema.Extensions["x-go-name"])
	assert.EqualT(t, "jsonClass", schema.Discriminator)

	sch := models["gazelle"]
	assert.Len(t, sch.AllOf, 2)
	cl, _ := sch.Extensions.GetString("x-class")
	assert.EqualT(t, "a.b.c.d.E", cl)
	cl, _ = sch.Extensions.GetString("x-go-name")
	assert.EqualT(t, "Gazelle", cl)

	sch = models["giraffe"]
	assert.Len(t, sch.AllOf, 2)
	cl, _ = sch.Extensions.GetString("x-class")
	assert.Empty(t, cl)
	cl, _ = sch.Extensions.GetString("x-go-name")
	assert.EqualT(t, "Giraffe", cl)

	// sch = noModelDefs["lion"]

	// b, _ := json.MarshalIndent(sch, "", "  ")
	// fmt.Println(string(b))

	scantest.CompareOrDumpJSON(t, models, "classification_schema_struct_discriminators.json")
}

func TestInterfaceDiscriminators(t *testing.T) {
	ctx := scantest.LoadClassificationPkgsCtx(t)
	models := make(map[string]oaispec.Schema)
	for _, tn := range []string{"BaseStruct", "Identifiable", "WaterType", "Fish", "TeslaCar", "ModelS", "ModelX", "ModelA", "Cars"} {
		decl := getClassificationModel(ctx, tn)
		require.NotNil(t, decl)

		prs := NewBuilder(ctx, decl)
		require.NoError(t, prs.Build(models))
	}

	schema, ok := models["fish"]

	if assert.TrueT(t, ok) && assert.Len(t, schema.AllOf, 5) {
		sch := schema.AllOf[3]
		assert.Len(t, sch.Properties, 1)
		scantest.AssertProperty(t, &sch, "string", "colorName", "", "ColorName")

		sch = schema.AllOf[2]
		assert.EqualT(t, "#/definitions/extra", sch.Ref.String())

		sch = schema.AllOf[0]
		assert.Len(t, sch.Properties, 1)
		scantest.AssertProperty(t, &sch, "integer", "id", "int64", "ID")

		sch = schema.AllOf[1]
		assert.EqualT(t, "#/definitions/water", sch.Ref.String())

		sch = schema.AllOf[4]
		assert.Len(t, sch.Properties, 2)
		scantest.AssertProperty(t, &sch, "string", "name", "", "Name")
		scantest.AssertProperty(t, &sch, "string", "jsonClass", "", "StructType")
		assert.EqualT(t, "jsonClass", sch.Discriminator)
	}

	schema, ok = models["modelS"]
	if assert.TrueT(t, ok) {
		assert.Len(t, schema.AllOf, 2)
		cl, _ := schema.Extensions.GetString("x-class")
		assert.EqualT(t, "com.tesla.models.ModelS", cl)
		cl, _ = schema.Extensions.GetString("x-go-name")
		assert.EqualT(t, "ModelS", cl)

		sch := schema.AllOf[0]
		assert.EqualT(t, "#/definitions/TeslaCar", sch.Ref.String())
		sch = schema.AllOf[1]
		assert.Len(t, sch.Properties, 1)
		scantest.AssertProperty(t, &sch, "string", "edition", "", "Edition")
	}

	schema, ok = models["modelA"]
	if assert.TrueT(t, ok) {
		cl, _ := schema.Extensions.GetString("x-go-name")
		assert.EqualT(t, "ModelA", cl)

		sch, ok := schema.Properties["Tesla"]
		if assert.TrueT(t, ok) {
			assert.EqualT(t, "#/definitions/TeslaCar", sch.Ref.String())
		}

		scantest.AssertProperty(t, &schema, "integer", "doors", "int64", "Doors")
	}

	scantest.CompareOrDumpJSON(t, models, "classification_schema_interface_discriminators.json")
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

func TestEmbeddedDescriptionAndTags(t *testing.T) {
	packagePattern := "./bugs/3125/minimal"
	packagePath := fixturesModule + "/bugs/3125/minimal"
	ctx, err := scanner.NewScanCtx(&scanner.Options{
		Packages:    []string{packagePattern},
		WorkDir:     scantest.FixturesDir(),
		DescWithRef: true,
	})
	require.NoError(t, err)
	decl, _ := ctx.FindDecl(packagePath, "Item")
	require.NotNil(t, decl)
	prs := NewBuilder(ctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["Item"]

	assert.Equal(t, []string{"value1", "value2"}, schema.Required)
	require.Len(t, schema.Properties, 2)

	require.MapContainsT(t, schema.Properties, "value1")
	assert.EqualT(t, "Nullable value", schema.Properties["value1"].Description)
	assert.Equal(t, true, schema.Properties["value1"].Extensions["x-nullable"])

	require.MapContainsT(t, schema.Properties, "value2")
	assert.EqualT(t, "Non-nullable value", schema.Properties["value2"].Description)
	assert.MapNotContainsT(t, schema.Properties["value2"].Extensions, "x-nullable")
	assert.Equal(t, `{"value": 42}`, schema.Properties["value2"].Example)

	scantest.CompareOrDumpJSON(t, models, "bugs_3125_schema.json")
}

func TestIssue2540(t *testing.T) {
	t.Run("should produce example and default for top level declaration only",
		testIssue2540(false, `{
		"Book": {
      "description": "At this moment, a book is only described by its publishing date\nand author.",
      "type": "object",
      "title": "Book holds all relevant information about a book.",
			"example": "{ \"Published\": 2026, \"Author\": \"Fred\" }",
      "default": "{ \"Published\": 1900, \"Author\": \"Unknown\" }",
      "properties": {
        "Author": {
          "$ref": "#/definitions/Author"
        },
        "Published": {
          "type": "integer",
          "format": "int64",
          "minimum": 0,
          "example": 2021
        }
      }
    }
  }`),
	)
	t.Run("should produce example and default for top level declaration and embedded $ref field",
		testIssue2540(true, `{
		"Book": {
      "description": "At this moment, a book is only described by its publishing date\nand author.",
      "type": "object",
      "title": "Book holds all relevant information about a book.",
			"example": "{ \"Published\": 2026, \"Author\": \"Fred\" }",
      "default": "{ \"Published\": 1900, \"Author\": \"Unknown\" }",
      "properties": {
        "Author": {
          "$ref": "#/definitions/Author",
          "example": "{ \"Name\": \"Tolkien\" }"
        },
        "Published": {
          "type": "integer",
          "format": "int64",
          "minimum": 0,
          "example": 2021
        }
      }
    }
  }`),
	)
}

func testIssue2540(descWithRef bool, expectedJSON string) func(*testing.T) {
	return func(t *testing.T) {
		packagePattern := "./bugs/2540/foo"
		packagePath := fixturesModule + "/bugs/2540/foo"
		ctx, err := scanner.NewScanCtx(&scanner.Options{
			Packages:       []string{packagePattern},
			WorkDir:        scantest.FixturesDir(),
			DescWithRef:    descWithRef,
			SkipExtensions: true,
		})
		require.NoError(t, err)

		decl, _ := ctx.FindDecl(packagePath, "Book")
		require.NotNil(t, decl)
		prs := NewBuilder(ctx, decl)
		models := make(map[string]oaispec.Schema)
		require.NoError(t, prs.Build(models))

		b, err := json.Marshal(models)
		require.NoError(t, err)
		assert.JSONEqT(t, expectedJSON, string(b))
	}
}
