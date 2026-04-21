// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/go-openapi/testify/v2/assert"

	oaispec "github.com/go-openapi/spec"
)

// enableSpecOutput toggles YAML dumping of generated specs for debugging.
const enableSpecOutput = false

// fixturesModule is the module path of the fixtures nested module.
const fixturesModule = "github.com/go-openapi/codescan/fixtures"

func marshalToYAMLFormat(swspec any) ([]byte, error) {
	b, err := json.Marshal(swspec)
	if err != nil {
		return nil, err
	}

	var jsonObj any
	if err := yaml.Unmarshal(b, &jsonObj); err != nil {
		return nil, err
	}

	return yaml.Marshal(jsonObj)
}

func assertHasExtension(t *testing.T, sch oaispec.Schema, ext string) {
	t.Helper()
	pkg, hasExt := sch.Extensions.GetString(ext)
	assert.TrueT(t, hasExt)
	assert.NotEmpty(t, pkg)
}

func assertHasGoPackageExt(t *testing.T, sch oaispec.Schema) {
	t.Helper()
	assertHasExtension(t, sch, "x-go-package")
}

func assertHasTitle(t *testing.T, sch oaispec.Schema) {
	t.Helper()
	assert.NotEmpty(t, sch.Title)
}

func assertHasNoTitle(t *testing.T, sch oaispec.Schema) {
	t.Helper()
	assert.Empty(t, sch.Title)
}

func assertIsRef(t *testing.T, schema *oaispec.Schema, fragment string) {
	t.Helper()

	assert.EqualT(t, fragment, schema.Ref.String())
}
