// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scantest

import (
	"fmt"
	"os"

	"github.com/go-openapi/codescan/internal/scanner"
)

// Loader names one of the ways a scan can get its package graph.
//
// The suites emit the same document under all three - that is what
// internal/integration/loader_agreement_test.go exists to hold - so which one runs is a question
// about what the run COSTS, and nothing else.
type Loader string

const (
	// LoaderSource reads every dependency from source, which is what a caller gets by default.
	LoaderSource Loader = "source"

	// LoaderCompiled takes dependency types from the compiler's export data.
	//
	// The suite is the case that option was made for: several hundred scans of small trees over one
	// dependency closure, so the closure is compiled once and every scan after the first reads it
	// back. Measured over internal/integration on a 16-core machine, that is 2m51s of CPU against
	// 58s - and a CI runner with two cores spends that difference on the wall.
	LoaderCompiled Loader = "compiled"

	// LoaderOwn resolves the graph with codescan's own loader, invoking no go command.
	//
	// Between the two on cost (1m50s of CPU on the same measurement), and the only one that exercises
	// the toolchain-free path over the whole corpus rather than over the agreement A/B alone.
	LoaderOwn Loader = "own"
)

// LoaderEnv is the variable that selects the loader for a test run.
//
// Unset, the suites run under whatever defaultLoader the build selected - which is LoaderSource
// unless the testloader_compiled or testloader_own build tag was given. The variable is the knob for
// a person at a terminal; the build tags are the knob for CI, which reaches `go test` through a
// shared workflow that forwards flags and not the environment.
const LoaderEnv = "CODESCAN_TEST_LOADER"

// TestLoader reports which loader the suites run under.
//
// An unrecognised value is a mistake worth stopping for rather than ignoring: a typo would otherwise
// silently measure and exercise the default while the operator believed otherwise.
func TestLoader() Loader {
	switch v := Loader(os.Getenv(LoaderEnv)); v {
	case "":
		return defaultLoader
	case LoaderSource, LoaderCompiled, LoaderOwn:
		return v
	default:
		panic(fmt.Sprintf("scantest: %s=%q is not one of %q, %q, %q",
			LoaderEnv, string(v), LoaderSource, LoaderCompiled, LoaderOwn))
	}
}

// ApplyLoader writes the selected loader onto opts and returns it, for chaining onto a call.
//
// It is deliberately a WRITE rather than a default: a test that has an opinion about how its graph
// is loaded must not route through here at all, because there is no way to tell a field left at its
// zero value from one set to it on purpose. Those tests call codescan.Run directly, and say so.
//
// Options.FS is the one thing it will not fight: a virtual filesystem forces the toolchain-free
// loader whatever anyone asks for, so asking for something else here would only produce a
// diagnostic about an intent that could not be met.
func ApplyLoader(opts *scanner.Options) *scanner.Options {
	if opts == nil || opts.FS != nil {
		return opts
	}

	switch selected := TestLoader(); selected {
	case LoaderCompiled:
		opts.CompiledDependencies = true
	case LoaderOwn:
		opts.ToolchainFreeLoader = true
	case LoaderSource:
		// The shipped default: nothing to set.
	default:
		// Unreachable, since TestLoader refuses anything else. Panicking rather than falling through
		// keeps it that way: a loader added to the type and forgotten here would otherwise quietly run
		// the whole suite under the default while reporting itself as something else.
		panic(fmt.Sprintf("scantest: no rule for loader %q", string(selected)))
	}

	return opts
}
