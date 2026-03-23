package resolvers

import (
	"testing"

	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestAddExtension(t *testing.T) {
	ve := &oaispec.VendorExtensible{
		Extensions: make(oaispec.Extensions),
	}

	key := "x-go-name"
	value := "Name"
	AddExtension(ve, key, value, false)
	veStr, ok := ve.Extensions[key].(string)
	require.TrueT(t, ok)
	assert.EqualT(t, value, veStr)

	key2 := "x-go-package"
	value2 := "schema"
	AddExtension(ve, key2, value2, false)
	veStr2, ok := ve.Extensions[key2].(string)
	require.TrueT(t, ok)
	assert.EqualT(t, value2, veStr2)

	key3 := "x-go-class"
	value3 := "Spec"
	AddExtension(ve, key3, value3, true)
	assert.Nil(t, ve.Extensions[key3])
}
