// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import (
	"flag"

	"github.com/go-openapi/codescan/internal/scanner"
)

// Options is the scan configuration these flags write to.
//
// It is codescan.Options under its internal name - the public type is an alias of this one - so this
// package depends downward on the scanner rather than back on the root package. A caller holding a
// *codescan.Options may pass it here unconverted: they are the same type.
type Options = scanner.Options

// option is one flag over a value-typed field: what it is called, what it defaults to, what it says
// for itself, and where it lands.
//
// The setter is the entry's identity. Naming the field in a string instead would let a rename pass
// the compiler and leave the flag writing nowhere, and it is what lets the coverage guard ask which
// field an entry actually touches rather than deriving a name from a name.
type option[T any] struct {
	name  string
	value T
	help  string
	field func(*Options) *T
}

// boolOptions is every boolean knob, grouped as the help text lists them.
//
// Defaults are the library's own (false) except where both existing commands already chose
// otherwise: -scan-models is on, because a command asked to produce a specification and handed a
// package of annotated models should produce their definitions.
var boolOptions = []option[bool]{ //nolint:gochecknoglobals // the flag table, read once at startup
	{
		"scan-models", true, "also emit definitions for swagger:model types",
		func(o *Options) *bool { return &o.ScanModels },
	},
	{
		"prune-unused-models", false, "drop discovered models nothing references (needs -scan-models)",
		func(o *Options) *bool { return &o.PruneUnusedModels },
	},
	{
		"exclude-deps", false, "scan only the packages matched by the patterns, never their dependencies",
		func(o *Options) *bool { return &o.ExcludeDeps },
	},

	{
		"ref-aliases", false, "aliases produce a $ref instead of being expanded",
		func(o *Options) *bool { return &o.RefAliases },
	},
	{
		"transparent-aliases", false, "aliases dissolve entirely, never producing a definition",
		func(o *Options) *bool { return &o.TransparentAliases },
	},

	{
		"emit-ref-siblings", false, "emit a $ref'd field's description and extensions beside the $ref",
		func(o *Options) *bool { return &o.EmitRefSiblings },
	},
	{
		"skip-all-of-compounding", false, "never wrap a $ref in an allOf compound",
		func(o *Options) *bool { return &o.SkipAllOfCompounding },
	},
	{
		"default-all-of-for-embeds", false, "render a plain struct embed as allOf composition",
		func(o *Options) *bool { return &o.DefaultAllOfForEmbeds },
	},

	{
		"set-x-nullable-for-pointers", false, "mark pointer fields x-nullable",
		func(o *Options) *bool { return &o.SetXNullableForPointers },
	},
	{
		"skip-extensions", false, "omit the x-go-* vendor extensions",
		func(o *Options) *bool { return &o.SkipExtensions },
	},
	{
		"emit-x-go-type", false, "stamp x-go-type on every emitted definition",
		func(o *Options) *bool { return &o.EmitXGoType },
	},

	{
		"single-line-comment-as-description", false, "a one-line doc comment is a description, never a title",
		func(o *Options) *bool { return &o.SingleLineCommentAsDescription },
	},
	{
		"after-decl-comments", false, "accept annotations inside or after a declaration",
		func(o *Options) *bool { return &o.AfterDeclComments },
	},
	{
		"clean-go-doc", false, "rewrite godoc-only syntax carried into a title or description",
		func(o *Options) *bool { return &o.CleanGoDoc },
	},
	{
		"skip-enum-descriptions", false, "keep the enum const-name mapping off the description",
		func(o *Options) *bool { return &o.SkipEnumDescriptions },
	},

	{
		"skip-jsonify-interface-methods", false, "emit interface-method names verbatim",
		func(o *Options) *bool { return &o.SkipJSONifyInterfaceMethods },
	},
	{
		"emit-hierarchical-names", false, "emit over-budget collision groups as nested definitions",
		func(o *Options) *bool { return &o.EmitHierarchicalNames },
	},

	{
		"stub-stdlib", false,
		"synthesize standard-library types instead of reading GOROOT: no Go installation and no module\n" +
			"cache needed, much smaller footprint, but not failsafe -- see the package documentation",
		func(o *Options) *bool { return &o.StubStdlib },
	},
	{
		"skip-compiled-dependencies", false,
		"read every dependency from source rather than taking its types from the compiler's export\n" +
			"data: the spec is the same either way, but this is much slower on a warm cache and much\n" +
			"faster on a cold one. Native builds only -- it needs -loader=go, which WebAssembly cannot run",
		func(o *Options) *bool { return &o.SkipCompiledDependencies },
	},
}

// stringOptions is every knob taking one string.
//
// The go environment lives here rather than being inherited, so that a scan is reproducible: each of
// these decides what gets built, and so what the specification says.
var stringOptions = []option[string]{ //nolint:gochecknoglobals // the flag table, read once at startup
	{
		"workdir", ".", "directory the scan runs from; patterns are relative to it",
		func(o *Options) *string { return &o.WorkDir },
	},
	{
		"build-tags", "", "comma-separated go build tags to apply while loading",
		func(o *Options) *string { return &o.BuildTags },
	},

	{
		"goos", "", "GOOS the scanned code is built for (default: this machine's)",
		func(o *Options) *string { return &o.GOOS },
	},
	{
		"goarch", "", "GOARCH the scanned code is built for (default: this machine's)",
		func(o *Options) *string { return &o.GOARCH },
	},
	{
		"goflags", "",
		`default go command flags, as GOFLAGS (e.g. "-tags=integration"); -build-tags wins`,
		func(o *Options) *string { return &o.GOFLAGS },
	},
	{
		"gowork", "",
		`workspace selection, as GOWORK: "off" to disable, a path to a go.work, empty to search upwards`,
		func(o *Options) *string { return &o.GOWORK },
	},
	{
		"goexperiment", "",
		`toolchain experiments to enable, as GOEXPERIMENT (e.g. "jsonv2")`,
		func(o *Options) *string { return &o.GOEXPERIMENT },
	},
}

// floatOptions is every knob taking one number.
var floatOptions = []option[float64]{ //nolint:gochecknoglobals // the flag table, read once at startup
	{
		"name-concat-budget", 0,
		"readability cutoff for collision-renaming by concatenation (0 = codescan's default of 0.65)",
		func(o *Options) *float64 { return &o.NameConcatBudget },
	},
}

// listOption is a flag taking a comma-separated list.
//
// threeWay says the field distinguishes "not given" from "given as empty". Where it is false the two
// collapse onto nil, which is what the scanner reads as "no filter" - so passing -include= is the
// same as not passing it at all, and means what it looks like it means.
type listOption struct {
	name     string
	help     string
	field    func(*Options) *[]string
	threeWay bool
}

// listOptions is every knob taking a list.
var listOptions = []listOption{ //nolint:gochecknoglobals // the flag table, read once at startup
	{
		name:  "include",
		help:  "comma-separated patterns; only matching packages are scanned",
		field: func(o *Options) *[]string { return &o.Include },
	},
	{
		name:  "exclude",
		help:  "comma-separated patterns; matching packages are skipped",
		field: func(o *Options) *[]string { return &o.Exclude },
	},
	{
		name:  "include-tags",
		help:  "comma-separated swagger tags; only matching operations are emitted",
		field: func(o *Options) *[]string { return &o.IncludeTags },
	},
	{
		name:  "exclude-tags",
		help:  "comma-separated swagger tags; matching operations are skipped",
		field: func(o *Options) *[]string { return &o.ExcludeTags },
	},
	{
		name: "name-from-tags",
		help: `ordered struct tags a field's name derives from (default "json"; pass empty to use the Go` + "\n" +
			"field name)",
		field:    func(o *Options) *[]string { return &o.NameFromTags },
		threeWay: true,
	},
}

// Values is where a parsed command line lands, before [Values.Apply] copies it onto the options.
//
// Lists are held as the raw string the caller typed: splitting at Apply time is what leaves the
// three-way cases able to tell an empty list from an absent one.
type Values struct {
	set *flag.FlagSet

	bools   map[string]*bool
	strings map[string]*string
	floats  map[string]*float64
	lists   map[string]*string
	loader  *string
}

// Register declares every flag on fs and returns where the parsed values will land.
//
// The whole surface is registered: a command that hides a knob only makes it unreachable, and the
// caller finds out by meeting "flag provided but not defined" after writing something against a
// surface that was never there.
func Register(fs *flag.FlagSet) *Values {
	values := &Values{
		set:     fs,
		bools:   make(map[string]*bool, len(boolOptions)),
		strings: make(map[string]*string, len(stringOptions)),
		floats:  make(map[string]*float64, len(floatOptions)),
		lists:   make(map[string]*string, len(listOptions)),
	}

	for _, opt := range boolOptions {
		values.bools[opt.name] = fs.Bool(opt.name, opt.value, opt.help)
	}
	for _, opt := range stringOptions {
		values.strings[opt.name] = fs.String(opt.name, opt.value, opt.help)
	}
	for _, opt := range floatOptions {
		values.floats[opt.name] = fs.Float64(opt.name, opt.value, opt.help)
	}
	for _, opt := range listOptions {
		values.lists[opt.name] = fs.String(opt.name, "", opt.help)
	}
	values.loader = fs.String(loaderFlag, loaderAuto, loaderHelp)

	return values
}

// Apply copies what was parsed onto the options a scan runs with.
//
// It fails only on a flag whose value is not one of a closed set (-loader): everything else was
// already validated by the flag package when it parsed.
func (v *Values) Apply(opts *Options) error {
	for _, opt := range boolOptions {
		if got, ok := v.bools[opt.name]; ok {
			*opt.field(opts) = *got
		}
	}
	for _, opt := range stringOptions {
		if got, ok := v.strings[opt.name]; ok {
			*opt.field(opts) = *got
		}
	}
	for _, opt := range floatOptions {
		if got, ok := v.floats[opt.name]; ok {
			*opt.field(opts) = *got
		}
	}
	for _, opt := range listOptions {
		got, ok := v.lists[opt.name]
		if !ok {
			continue
		}
		*opt.field(opts) = v.list(opt, *got)
	}

	own, err := resolveLoader(*v.loader)
	if err != nil {
		return err
	}
	opts.ToolchainFreeLoader = own

	return nil
}

// list reads one list flag, honouring the three-way contract where the field has one.
//
//   - nil (unset) keeps whatever the library does by default
//   - an empty but non-nil slice is an explicit "consult nothing"
//
// Collapsing those two would make -name-from-tags= silently mean the opposite of what it says: not
// "use the Go field name" but "use the json tag".
func (v *Values) list(opt listOption, raw string) []string {
	if !opt.threeWay {
		return SplitList(raw)
	}

	if !v.passed(opt.name) {
		return nil
	}
	if parsed := SplitList(raw); parsed != nil {
		return parsed
	}

	return []string{}
}

// passed reports whether the named flag appeared on the command line, as opposed to holding its
// default.
func (v *Values) passed(name string) bool {
	seen := false
	v.set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			seen = true
		}
	})

	return seen
}
