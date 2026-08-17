// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package loadertest lets the runnable examples answer to the run-wide loader setting.
//
// The examples are documentation that happens to be tested, so their own source says nothing about
// how a package graph is fetched - a reader copying one wants the ordinary defaults. Their TESTS are
// part of the same run as everything else, though, and pay the same cost, so the scan each one makes
// goes through here.
//
// See internal/testloader for what selects the loader and why.
package loadertest

import (
	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/testloader"
)

// Apply writes the loader this run was asked for onto opts, and returns it for chaining.
//
// A test that pins a loader must not route through here - there is no telling a field left at its
// zero value from one set on purpose - which is why this refuses one rather than writing over it.
func Apply(opts *codescan.Options) *codescan.Options {
	if opts == nil || opts.FS != nil {
		return opts
	}
	if opts.CompiledDependencies || opts.ToolchainFreeLoader {
		panic("loadertest: these options already pin a loader, so they must not be routed through " +
			"Apply; drop the call and leave the pin, or drop the pin and keep the call")
	}

	switch selected := testloader.Selected(); selected {
	case testloader.LoaderCompiled:
		opts.CompiledDependencies = true
	case testloader.LoaderOwn:
		opts.ToolchainFreeLoader = true
	case testloader.LoaderSource:
	default:
		panic("loadertest: no rule for loader " + string(selected))
	}

	return opts
}
