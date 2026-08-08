// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build baseline

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-openapi/codescan"
)

// extraOptions is the baseline build's stub: the released library this build links against has one
// way to load and scan, and no knob to select another.
//
// It still accepts the flags, so run.sh can drive both builds from one command line, and it exits
// rather than ignoring them. A baseline binary that quietly accepted -compiled-deps would emit a
// default-configuration measurement under a label saying otherwise — an error invisible in the
// results, which is exactly the kind this harness exists to avoid.
type extraOptions struct {
	compiledDependencies bool
	toolchainFreeLoader  bool
}

func (e *extraOptions) register() {
	flag.BoolVar(&e.compiledDependencies, "compiled-deps", false,
		"not available in the baseline build")
	flag.BoolVar(&e.toolchainFreeLoader, "toolchain-free", false,
		"not available in the baseline build")
}

func (e *extraOptions) apply(_ *codescan.Options) {
	for name, set := range map[string]bool{
		"-compiled-deps":  e.compiledDependencies,
		"-toolchain-free": e.toolchainFreeLoader,
	} {
		if set {
			fmt.Fprintf(os.Stderr,
				"loader-benchmark: %s does not exist in the baseline release; there is nothing to compare\n", name)
			os.Exit(2)
		}
	}
}
