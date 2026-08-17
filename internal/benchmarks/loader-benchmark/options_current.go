// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !baseline

package main

import (
	"flag"

	"github.com/go-openapi/codescan"
)

// extraOptions carries the loader knobs that only exist in the current tree.
//
// The whole point of this harness is to compile one source against two versions of the library, so
// every option younger than the baseline release has to sit behind this seam. The baseline build gets
// the stub in options_baseline.go, which accepts the same flags and refuses to pretend they did
// anything — silently ignoring them would report a default-configuration run under a label claiming
// otherwise, which is the one measurement error that cannot be spotted in the output.
type extraOptions struct {
	compiledDependencies bool
	toolchainFreeLoader  bool
}

func (e *extraOptions) register() {
	flag.BoolVar(&e.compiledDependencies, "compiled-deps", false,
		"take dependency types from compiled export data")
	flag.BoolVar(&e.toolchainFreeLoader, "toolchain-free", false,
		"load and type-check without invoking the Go toolchain")
}

// apply sets the configuration this run measures.
//
// -compiled-deps keeps its meaning across every build measured — "take dependency types from compiled
// export data" — whichever way the field behind it was spelled at that release. So the flag selects a
// configuration rather than reporting the library's default.
func (e *extraOptions) apply(opts *codescan.Options) {
	opts.CompiledDependencies = e.compiledDependencies
	opts.ToolchainFreeLoader = e.toolchainFreeLoader
}
