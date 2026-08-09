// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/go-openapi/codescan"
)

// boolOptions is every boolean knob on codescan.Options that this command exposes, in the order the
// help text lists them.
//
// A table rather than a field per flag: the option set grows, and a caller that cannot see a knob has
// no way to reach it. TestFlagsCoverEveryBoolOption fails when a new one lands without a decision
// here, so the gap is caught at the source rather than by a user meeting
// "flag provided but not defined".
//
// Flag names are the field name in kebab-case, without exception - guessing a shorter spelling is how
// a caller ends up guessing wrong.
var boolOptions = []boolOption{ //nolint:gochecknoglobals // the flag table, read once at startup
	{
		"scan-models", true, "also emit definitions for swagger:model types",
		func(o *codescan.Options) *bool { return &o.ScanModels },
	},
	{
		"prune-unused-models", false, "drop discovered models nothing references (needs -scan-models)",
		func(o *codescan.Options) *bool { return &o.PruneUnusedModels },
	},
	{
		"exclude-deps", false, "scan only the packages matched by the patterns, never their dependencies",
		func(o *codescan.Options) *bool { return &o.ExcludeDeps },
	},

	{
		"ref-aliases", false, "aliases produce a $ref instead of being expanded",
		func(o *codescan.Options) *bool { return &o.RefAliases },
	},
	{
		"transparent-aliases", false, "aliases dissolve entirely, never producing a definition",
		func(o *codescan.Options) *bool { return &o.TransparentAliases },
	},

	{
		"emit-ref-siblings", false, "emit a $ref'd field's description and extensions beside the $ref",
		func(o *codescan.Options) *bool { return &o.EmitRefSiblings },
	},
	{
		"skip-all-of-compounding", false, "never wrap a $ref in an allOf compound",
		func(o *codescan.Options) *bool { return &o.SkipAllOfCompounding },
	},
	{
		"default-all-of-for-embeds", false, "render a plain struct embed as allOf composition",
		func(o *codescan.Options) *bool { return &o.DefaultAllOfForEmbeds },
	},

	{
		"set-x-nullable-for-pointers", false, "mark pointer fields x-nullable",
		func(o *codescan.Options) *bool { return &o.SetXNullableForPointers },
	},
	{
		"skip-extensions", false, "omit the x-go-* vendor extensions",
		func(o *codescan.Options) *bool { return &o.SkipExtensions },
	},
	{
		"emit-x-go-type", false, "stamp x-go-type on every emitted definition",
		func(o *codescan.Options) *bool { return &o.EmitXGoType },
	},

	{
		"single-line-comment-as-description", false, "a one-line doc comment is a description, never a title",
		func(o *codescan.Options) *bool { return &o.SingleLineCommentAsDescription },
	},
	{
		"after-decl-comments", false, "accept annotations inside or after a declaration",
		func(o *codescan.Options) *bool { return &o.AfterDeclComments },
	},
	{
		"clean-go-doc", false, "rewrite godoc-only syntax carried into a title or description",
		func(o *codescan.Options) *bool { return &o.CleanGoDoc },
	},
	{
		"skip-enum-descriptions", false, "keep the enum const-name mapping off the description",
		func(o *codescan.Options) *bool { return &o.SkipEnumDescriptions },
	},

	{
		"skip-jsonify-interface-methods", false, "emit interface-method names verbatim",
		func(o *codescan.Options) *bool { return &o.SkipJSONifyInterfaceMethods },
	},
	{
		"emit-hierarchical-names", false, "emit over-budget collision groups as nested definitions",
		func(o *codescan.Options) *bool { return &o.EmitHierarchicalNames },
	},

	{
		"stub-stdlib", false,
		"synthesize standard-library types instead of reading GOROOT: no Go installation and no module\n" +
			"cache needed, much smaller footprint, but not failsafe -- see the package documentation",
		func(o *codescan.Options) *bool { return &o.StubStdlib },
	},
	{
		"skip-compiled-dependencies", false,
		"read every dependency from source rather than taking its types from the compiler's export\n" +
			"data: the spec is the same either way, but this is much slower on a warm cache and much\n" +
			"faster on a cold one. Native builds only -- it needs -loader=go, which WebAssembly cannot run",
		func(o *codescan.Options) *bool { return &o.SkipCompiledDependencies },
	},
}

type boolOption struct {
	name  string
	value bool
	help  string
	field func(*codescan.Options) *bool
}

// registerBools declares every boolean flag and returns where the parsed values land.
func registerBools(fs *flag.FlagSet) map[string]*bool {
	parsed := make(map[string]*bool, len(boolOptions))
	for _, opt := range boolOptions {
		parsed[opt.name] = fs.Bool(opt.name, opt.value, opt.help)
	}

	return parsed
}

// applyBools copies the parsed values onto the options the scan runs with.
func applyBools(opts *codescan.Options, parsed map[string]*bool) {
	for _, opt := range boolOptions {
		if got, ok := parsed[opt.name]; ok {
			*opt.field(opts) = *got
		}
	}
}
