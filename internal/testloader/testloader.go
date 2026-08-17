// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package testloader carries the one setting that decides which loader a test run uses.
//
// It depends on nothing. That is the whole reason it exists as a package of its own: the scanner's
// own tests are in-package, and scantest imports the scanner, so a policy living there could never
// be read from where most of the scans are.
//
// See internal/scantest for how it is applied, and for what it buys.
package testloader

import (
	"fmt"
	"os"
)

// Loader names one of the ways a scan can get its package graph.
//
// The suites emit the same document under all three, so which one runs is a question about what the
// run COSTS and nothing else.
type Loader string

const (
	// LoaderSource reads every dependency from source, which is what a caller gets by default.
	LoaderSource Loader = "source"

	// LoaderCompiled takes dependency types from the compiler's export data.
	LoaderCompiled Loader = "compiled"

	// LoaderOwn resolves the graph with codescan's own loader, invoking no go command.
	LoaderOwn Loader = "own"
)

// Env is the variable that selects the loader for a test run.
//
// Unset, the suites run under whatever the build selected - LoaderSource unless the
// testloader_compiled or testloader_own build tag was given. The variable is the knob for a person
// at a terminal; the build tags are the knob for CI, which reaches `go test` through a shared
// workflow that forwards flags and not the environment.
const Env = "CODESCAN_TEST_LOADER"

// Selected reports which loader the suites run under.
//
// An unrecognised value stops the run rather than being ignored: a typo would otherwise silently
// exercise the default while the operator believed otherwise, which has already cost a CI round.
func Selected() Loader {
	switch v := Loader(os.Getenv(Env)); v {
	case "":
		return defaultLoader
	case LoaderSource, LoaderCompiled, LoaderOwn:
		return v
	default:
		panic(fmt.Sprintf("testloader: %s=%q is not one of %q, %q, %q",
			Env, string(v), LoaderSource, LoaderCompiled, LoaderOwn))
	}
}

// Describe is a one-line statement of which loader is in force.
//
// A run is otherwise indistinguishable from a run configured differently: the tag that selects the
// loader is set in a workflow file, far from any test, and two rounds of CI debugging went on
// inferring exactly that from wall-clock time.
//
// Returned rather than printed, and reported from an ordinary test rather than from TestMain,
// because TestMain also runs under `go test -list`: the fuzz matrix builds itself by treating every
// listed line as a target name, and a stray line there becomes a fuzz test that does not exist.
func Describe(suite string) string {
	return fmt.Sprintf("codescan %s suite: loader=%s (set %s or -tags=testloader_<name>)",
		suite, Selected(), Env)
}
