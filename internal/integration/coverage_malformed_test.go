// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"
)

// Bucket-B error-path tests. Each subfixture under fixtures/enhancements/
// malformed/ carries exactly one annotation that the scanner cannot
// reconcile, so Run() must return a non-nil error. No goldens are
// produced — these tests exist purely to pin the error surface.

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

func TestMalformed_MetaBadExtensionKey(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/meta-bad-ext-key/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

func TestMalformed_InfoBadExtensionKey(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/info-bad-ext-key/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

func TestMalformed_BadContact(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/bad-contact/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

func TestMalformed_DuplicateBodyTag(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/duplicate-body-tag/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

func TestMalformed_BadResponseTag(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/bad-response-tag/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}

func TestMalformed_BadSecurityDefinitions(t *testing.T) {
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/malformed/bad-sec-defs/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.Error(t, err)
}
