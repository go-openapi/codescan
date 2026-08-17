// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/go-openapi/testify/v2/assert"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
)

// runScan is codescan.Run under whichever loader the suite was asked to run with.
//
// Almost every test here is about what a scan EMITS, and the emitted document is the same however
// the package graph was fetched - held over the whole corpus by TestLoaderAgreement. What differs is
// the cost, and this suite is where the cost lands: several hundred scans of small trees over one
// dependency closure. Reading that closure from source each time is roughly three times the CPU of
// compiling it once and reading it back, which a runner with two cores pays for on the wall.
//
// So the configuration is a property of the RUN rather than of each test, and lives in one place.
// See scantest.LoaderEnv for how to select it.
//
// The tests that are about loading itself do not come through here: they set their own options and
// call codescan.Run directly, because a test that pins a loader cannot have one applied over it.
func runScan(opts *codescan.Options) (*oaispec.Swagger, error) {
	return codescan.Run(scantest.ApplyLoader(opts))
}

// enableSpecOutput toggles YAML dumping of generated specs for debugging.
const enableSpecOutput = false

// fixturesModule is the module path of the fixtures nested module.
const fixturesModule = "github.com/go-openapi/codescan/testdata"

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
