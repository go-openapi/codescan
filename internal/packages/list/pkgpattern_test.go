// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-BSD-go file.
//
// SPDX-License-Identifier: BSD-3-Clause

// Derived from cmd/internal/pkgpattern/pat_test.go at go1.26.5: the MatchPattern table and the little interpreter that
// reads it, with the tables for the entry points codescan does not use removed.
//
// This is the reason the matcher itself was copied.
// The table is the specification written down — every clause of "go help packages" that anyone has ever got wrong,
// including the three deep-vendor cases our own matcher failed before it was replaced.

package list

import (
	"strings"
	"testing"
)

var matchPatternTests = `
	pattern ...
	match foo

	pattern net
	match net
	not net/http

	pattern net/http
	match net/http
	not net

	pattern net...
	match net net/http netchan
	not not/http not/net/http

	# Special cases. Quoting docs:

	# First, /... at the end of the pattern can match an empty string,
	# so that net/... matches both net and packages in its subdirectories, like net/http.
	pattern net/...
	match net net/http
	not not/http not/net/http netchan

	# Second, any slash-separated pattern element containing a wildcard never
	# participates in a match of the "vendor" element in the path of a vendored
	# package, so that ./... does not match packages in subdirectories of
	# ./vendor or ./mycode/vendor, but ./vendor/... and ./mycode/vendor/... do.
	# Note, however, that a directory named vendor that itself contains code
	# is not a vendored package: cmd/vendor would be a command named vendor,
	# and the pattern cmd/... matches it.
	pattern ./...
	match ./vendor ./mycode/vendor
	not ./vendor/foo ./mycode/vendor/foo

	pattern ./vendor/...
	match ./vendor/foo ./vendor/foo/vendor
	not ./vendor/foo/vendor/bar

	pattern mycode/vendor/...
	match mycode/vendor mycode/vendor/foo mycode/vendor/foo/vendor
	not mycode/vendor/foo/vendor/bar

	pattern x/vendor/y
	match x/vendor/y
	not x/vendor

	pattern x/vendor/y/...
	match x/vendor/y x/vendor/y/z x/vendor/y/vendor x/vendor/y/z/vendor
	not x/vendor/y/vendor/z

	pattern .../vendor/...
	match x/vendor/y x/vendor/y/z x/vendor/y/vendor x/vendor/y/z/vendor
`

func TestMatchPattern(t *testing.T) {
	testPatterns(t, "MatchPattern", matchPatternTests, func(pattern, name string) bool {
		return MatchPattern(pattern)(name)
	})
}

func testPatterns(t *testing.T, name, tests string, fn func(string, string) bool) {
	var patterns []string
	for _, line := range strings.Split(tests, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		default:
			t.Fatalf("unknown directive %q", f[0])
		case "pattern":
			patterns = f[1:]
		case "match", "not":
			want := f[0] == "match"
			for _, pattern := range patterns {
				for _, in := range f[1:] {
					if fn(pattern, in) != want {
						t.Errorf("%s(%q, %q) = %v, want %v", name, pattern, in, !want, want)
					}
				}
			}
		}
	}
}
