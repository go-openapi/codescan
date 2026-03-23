// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"
)

// These tests are the migration canary ported from go-swagger's
// cmd/swagger/commands/generate/spec_test.go. They exercise the same api.go
// fixture with the three Options variants go-swagger tests, and compare
// against go-swagger's pre-existing golden JSON files. If these pass, codescan
// still produces the specs go-swagger has depended on for years.

func TestGoSwagger_GenerateJSONSpec(t *testing.T) {
	opts := codescan.Options{
		WorkDir:  filepath.Join(scantest.FixturesDir(), "goparsing", "spec"),
		Packages: []string{"./..."},
	}
	swspec, err := codescan.Run(&opts)
	require.NoError(t, err)

	scantest.CompareOrDumpJSON(t, swspec, "api_spec_go111.json")
}

func TestGoSwagger_GenerateJSONSpec_RefAliases(t *testing.T) {
	opts := codescan.Options{
		WorkDir:    filepath.Join(scantest.FixturesDir(), "goparsing", "spec"),
		Packages:   []string{"./..."},
		RefAliases: true,
	}
	swspec, err := codescan.Run(&opts)
	require.NoError(t, err)

	scantest.CompareOrDumpJSON(t, swspec, "api_spec_go111_ref.json")
}

func TestGoSwagger_GenerateJSONSpec_TransparentAliases(t *testing.T) {
	opts := codescan.Options{
		WorkDir:            filepath.Join(scantest.FixturesDir(), "goparsing", "spec"),
		Packages:           []string{"./..."},
		TransparentAliases: true,
	}
	swspec, err := codescan.Run(&opts)
	require.NoError(t, err)

	scantest.CompareOrDumpJSON(t, swspec, "api_spec_go111_transparent.json")
}
