// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The playground declares the -format=json envelope a second time, in TypeScript, because a browser
// cannot read our struct tags. Nothing connects the two: renaming a field here leaves the app
// compiling, type-checking and rendering undefined where a value used to be, and the front-end
// checks cannot see it because they never read Go.
//
// So the producer holds the test. It runs in the ordinary Go suite, on the pull request that
// introduces the drift, rather than waiting for somebody to open the page.
const typesTS = "../../hack/doc-site/genspec-wasi/src/lib/types.ts"

// tsTypes are the declarations mirroring this package's JSON, by Go type.
//
// ScanOptions is deliberately absent: it mirrors the command's *flags*, not this envelope, and
// options_test.go already guards that surface from the Go side.
func tsTypes() map[string]any {
	return map[string]any{
		"Envelope":     envelope{},
		"Diagnostic":   jsonDiag{},
		"Anchor":       jsonAnchor{},
		"RuntimeStats": jsonRuntime{},
	}
}

func TestEnvelopeMatchesTheTypeScriptContract(t *testing.T) {
	t.Parallel()

	declared := parseTSTypes(t, typesTS)

	for name, goValue := range tsTypes() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ts, ok := declared[name]
			require.Truef(t, ok, "%s declares no type %s; the envelope is mirrored there", typesTS, name)

			assert.Equalf(t, goFields(goValue), ts,
				"%s and the Go %T have drifted apart", name, goValue)
		})
	}
}

// goFields reads the JSON contract off a struct: name, and whether it may be absent.
func goFields(v any) map[string]bool {
	typ := reflect.TypeOf(v)
	fields := make(map[string]bool, typ.NumField())

	for i := range typ.NumField() {
		tag, ok := typ.Field(i).Tag.Lookup("json")
		if !ok || tag == "-" {
			continue
		}

		name, opts, _ := strings.Cut(tag, ",")
		fields[name] = strings.Contains(opts, "omitempty")
	}

	return fields
}

var (
	rxTSTypeHead = regexp.MustCompile(`^export type (\w+) = \{$`)
	// A property line, and only that: a comment, a blank line or the closing brace does not match.
	rxTSProperty = regexp.MustCompile(`^\s+(\w+)(\??):`)
)

// parseTSTypes reads the `export type X = { ... }` declarations, yielding each one's property names
// and whether the property is optional.
//
// A line scan rather than a TypeScript parse: the file is hand-written to one shape, and a test that
// needed a parser to state a contract would be harder to trust than the contract.
func parseTSTypes(t *testing.T, path string) map[string]map[string]bool {
	t.Helper()

	source, err := os.ReadFile(filepath.FromSlash(path))
	require.NoErrorf(t, err, "cannot read the playground's type declarations at %s", path)

	types := make(map[string]map[string]bool)

	var current map[string]bool

	for line := range strings.SplitSeq(string(source), "\n") {
		switch {
		case current == nil:
			if head := rxTSTypeHead.FindStringSubmatch(line); head != nil {
				current = make(map[string]bool)
				types[head[1]] = current
			}
		case strings.HasPrefix(line, "};"):
			current = nil
		default:
			if prop := rxTSProperty.FindStringSubmatch(line); prop != nil {
				current[prop[1]] = prop[2] == "?"
			}
		}
	}

	require.NotEmptyf(t, types, "found no type declarations in %s", path)

	return types
}
