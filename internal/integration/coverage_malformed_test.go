// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"
)

// Bucket-B error-path tests.
// Each subfixture under fixtures/enhancements/ malformed/ carries exactly one annotation that the
// scanner cannot reconcile, so Run() must return a non-nil error.
// No goldens are produced — these tests exist purely to pin the error surface.

func TestMalformed_DefaultInt(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/default-int/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

func TestMalformed_ExampleInt(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/example-int/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

// TestMalformed_MetaBadExtensionKey was an error-returning test against the legacy meta
// validateExtensionNames path.
//
// M6.5-E aligns meta extension handling with routes — non-x-* keys emit a CodeInvalidAnnotation
// diagnostic at grammar parse time and drop, but Run still succeeds.
// The fixture's bad key ends up absent from the captured golden.
func TestMalformed_MetaBadExtensionKey(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/meta-bad-ext-key/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "malformed_meta_bad_ext_key.json")
}

// TestMalformed_InfoBadExtensionKey — see TestMalformed_MetaBadExtensionKey.
//
// Same diagnose-and-drop shift, here under the InfoExtensions: keyword.
func TestMalformed_InfoBadExtensionKey(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/info-bad-ext-key/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "malformed_info_bad_ext_key.json")
}

func TestMalformed_BadContact(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/bad-contact/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

// TestMalformed_DuplicateBodyTag was an error-returning test against the legacy routes body parser.
//
// M6.5-C shifts the routes body sub-language to a diagnose-and-continue contract (matching the rest
// of the grammar2 surface): malformed lines emit CodeInvalidAnnotation and the response is dropped,
// but Run still succeeds.
//
// The fixture's malformed response line ends up absent from the captured golden — the witness IS
// the dropped output.
func TestMalformed_DuplicateBodyTag(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/duplicate-body-tag/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "malformed_duplicate_body_tag.json")
}

// TestMalformed_BadResponseTag — see TestMalformed_DuplicateBodyTag.
//
// Unknown tag prefixes emit a diagnostic and drop the response line; the rest of the route builds
// normally.
func TestMalformed_BadResponseTag(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/bad-response-tag/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	scantest.CompareOrDumpJSON(t, doc, "malformed_bad_response_tag.json")
}

func TestMalformed_BadSecurityDefinitions(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/bad-sec-defs/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}
