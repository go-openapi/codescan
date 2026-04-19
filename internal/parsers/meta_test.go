// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/scantest/classification"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func TestSetInfoVersion(t *testing.T) {
	info := new(oaispec.Swagger)
	err := setInfoVersion(info, []string{"0.0.1"})
	require.NoError(t, err)
	assert.EqualT(t, "0.0.1", info.Info.Version)
}

func TestSetInfoLicense(t *testing.T) {
	info := new(oaispec.Swagger)
	err := setInfoLicense(info, []string{"MIT http://license.org/MIT"})
	require.NoError(t, err)
	assert.EqualT(t, "MIT", info.Info.License.Name)
	assert.EqualT(t, "http://license.org/MIT", info.Info.License.URL)
}

func TestSetInfoContact(t *testing.T) {
	info := new(oaispec.Swagger)
	err := setInfoContact(info, []string{"Homer J. Simpson <homer@simpsons.com> http://simpsons.com"})
	require.NoError(t, err)
	assert.EqualT(t, "Homer J. Simpson", info.Info.Contact.Name)
	assert.EqualT(t, "homer@simpsons.com", info.Info.Contact.Email)
	assert.EqualT(t, "http://simpsons.com", info.Info.Contact.URL)
}

func TestParseInfo(t *testing.T) {
	swspec := new(oaispec.Swagger)
	parser := NewMetaParser(swspec)
	docFile := "../../fixtures/goparsing/classification/doc.go"
	fileSet := token.NewFileSet()
	fileTree, err := goparser.ParseFile(fileSet, docFile, nil, goparser.ParseComments)
	if err != nil {
		t.FailNow()
	}

	err = parser.Parse(fileTree.Doc)

	require.NoError(t, err)
	classification.VerifyInfo(t, swspec.Info)
}

func TestParseSwagger(t *testing.T) {
	swspec := new(oaispec.Swagger)
	parser := NewMetaParser(swspec)
	docFile := "../../fixtures/goparsing/classification/doc.go"
	fileSet := token.NewFileSet()
	fileTree, err := goparser.ParseFile(fileSet, docFile, nil, goparser.ParseComments)
	if err != nil {
		t.FailNow()
	}

	err = parser.Parse(fileTree.Doc)
	verifyMeta(t, swspec)

	require.NoError(t, err)
}

func verifyMeta(t *testing.T, doc *oaispec.Swagger) {
	assert.NotNil(t, doc)
	classification.VerifyInfo(t, doc.Info)
	assert.Equal(t, []string{"application/json", "application/xml"}, doc.Consumes)
	assert.Equal(t, []string{"application/json", "application/xml"}, doc.Produces)
	assert.Equal(t, []string{"http", "https"}, doc.Schemes)
	assert.Equal(t, []map[string][]string{{"api_key": {}}}, doc.Security)
	expectedSecuritySchemaKey := oaispec.SecurityScheme{
		SecuritySchemeProps: oaispec.SecuritySchemeProps{
			Type: "apiKey",
			In:   "header",
			Name: "KEY",
		},
	}
	expectedSecuritySchemaOAuth := oaispec.SecurityScheme{
		SecuritySchemeProps: oaispec.SecuritySchemeProps{ //nolint:gosec // G101: false positive, test fixture not real credentials
			Type:             "oauth2",
			In:               "header",
			AuthorizationURL: "/oauth2/auth",
			TokenURL:         "/oauth2/token",
			Flow:             "accessCode",
			Scopes: map[string]string{
				"bla1": "foo1",
				"bla2": "foo2",
			},
		},
	}
	expectedExtensions := oaispec.Extensions{
		"x-meta-array": []any{
			"value1",
			"value2",
		},
		"x-meta-array-obj": []any{
			map[string]any{
				"name":  "obj",
				"value": "field",
			},
		},
		"x-meta-value": "value",
	}
	expectedInfoExtensions := oaispec.Extensions{
		"x-info-array": []any{
			"value1",
			"value2",
		},
		"x-info-array-obj": []any{
			map[string]any{
				"name":  "obj",
				"value": "field",
			},
		},
		"x-info-value": "value",
	}
	assert.NotNil(t, doc.SecurityDefinitions["api_key"])
	assert.NotNil(t, doc.SecurityDefinitions["oauth2"])
	assert.Equal(t, oaispec.SecurityDefinitions{"api_key": &expectedSecuritySchemaKey, "oauth2": &expectedSecuritySchemaOAuth}, doc.SecurityDefinitions)
	assert.Equal(t, expectedExtensions, doc.Extensions)
	assert.Equal(t, expectedInfoExtensions, doc.Info.Extensions)
	assert.EqualT(t, "localhost", doc.Host)
	assert.EqualT(t, "/v2", doc.BasePath)
}

func TestMoreParseMeta(t *testing.T) {
	for _, docFile := range []string{
		"../../fixtures/goparsing/meta/v1/doc.go",
		"../../fixtures/goparsing/meta/v2/doc.go",
		"../../fixtures/goparsing/meta/v3/doc.go",
		"../../fixtures/goparsing/meta/v4/doc.go",
	} {
		swspec := new(oaispec.Swagger)
		parser := NewMetaParser(swspec)
		fileSet := token.NewFileSet()
		fileTree, err := goparser.ParseFile(fileSet, docFile, nil, goparser.ParseComments)
		if err != nil {
			t.FailNow()
		}

		err = parser.Parse(fileTree.Doc)
		require.NoError(t, err)
		assert.EqualT(t, "there are no TOS at this moment, use at your own risk we take no responsibility", swspec.Info.TermsOfService)
		/*
			jazon, err := json.MarshalIndent(swoaispec.Info, "", " ")
			require.NoError(t, err)
			t.Logf("%v", string(jazon))
		*/
	}
}

func TestSetInfoVersion_Empty(t *testing.T) {
	swspec := new(oaispec.Swagger)
	require.NoError(t, setInfoVersion(swspec, nil))
	assert.Nil(t, swspec.Info)
}

func TestSetSwaggerHost_Empty(t *testing.T) {
	swspec := new(oaispec.Swagger)
	require.NoError(t, setSwaggerHost(swspec, nil))
	assert.EqualT(t, "localhost", swspec.Host) // fallback
	swspec2 := new(oaispec.Swagger)
	require.NoError(t, setSwaggerHost(swspec2, []string{""}))
	assert.EqualT(t, "localhost", swspec2.Host) // fallback
}

func TestSetInfoContact_Empty(t *testing.T) {
	swspec := new(oaispec.Swagger)
	require.NoError(t, setInfoContact(swspec, nil))
	assert.Nil(t, swspec.Info)
	require.NoError(t, setInfoContact(swspec, []string{""}))
}

func TestSetInfoContact_BadEmail(t *testing.T) {
	swspec := new(oaispec.Swagger)
	err := setInfoContact(swspec, []string{"not-a-valid-email-address <<<"})
	require.Error(t, err)
}

func TestSetInfoLicense_Empty(t *testing.T) {
	swspec := new(oaispec.Swagger)
	require.NoError(t, setInfoLicense(swspec, nil))
	assert.Nil(t, swspec.Info)
	require.NoError(t, setInfoLicense(swspec, []string{""}))
}

func TestSetMetaSingle_Parse_Empty(t *testing.T) {
	swspec := new(oaispec.Swagger)
	s := &setMetaSingle{Spec: swspec, Rx: rxVersion, Set: setInfoVersion}
	require.NoError(t, s.Parse(nil))
	require.NoError(t, s.Parse([]string{""}))
	// Line that doesn't match the regex
	require.NoError(t, s.Parse([]string{"no match here"}))
}

func TestSplitURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		line    string
		wantNot string
		wantURL string
	}{
		{"with http url", "MIT http://example.com", "MIT", "http://example.com"},
		{"with https url", "MIT https://example.com", "MIT", "https://example.com"},
		{"url only", "http://example.com", "", "http://example.com"},
		{"no url", "just text", "just text", ""},
		{"empty", "", "", ""},
		{"ws url", "live ws://example.com/ws", "live", "ws://example.com/ws"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			notURL, url := splitURL(tc.line)
			assert.EqualT(t, tc.wantNot, notURL)
			assert.EqualT(t, tc.wantURL, url)
		})
	}
}

func TestMetaVendorExtensibleSetter_InvalidKey(t *testing.T) {
	swspec := new(oaispec.Swagger)
	setter := metaVendorExtensibleSetter(swspec)
	// Extension key that doesn't start with x-
	err := setter([]byte(`{"not-x-key": "value"}`))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrParser)
}

func TestMetaVendorExtensibleSetter_BadJSON(t *testing.T) {
	swspec := new(oaispec.Swagger)
	setter := metaVendorExtensibleSetter(swspec)
	err := setter([]byte(`{bad json`))
	require.Error(t, err)
}

func TestInfoVendorExtensibleSetter_InvalidKey(t *testing.T) {
	swspec := &oaispec.Swagger{}
	swspec.Info = new(oaispec.Info)
	setter := infoVendorExtensibleSetter(swspec)
	err := setter([]byte(`{"invalid-key": "value"}`))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrParser)
}

func TestInfoVendorExtensibleSetter_BadJSON(t *testing.T) {
	swspec := &oaispec.Swagger{}
	swspec.Info = new(oaispec.Info)
	setter := infoVendorExtensibleSetter(swspec)
	err := setter([]byte(`{bad json`))
	require.Error(t, err)
}

func TestMetaSecurityDefinitionsSetter_BadJSON(t *testing.T) {
	swspec := new(oaispec.Swagger)
	setter := metaSecurityDefinitionsSetter(swspec)
	err := setter([]byte(`{bad json`))
	require.Error(t, err)
}
