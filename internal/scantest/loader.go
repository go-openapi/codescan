// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scantest

import (
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/testloader"
)

// Re-exported so a caller that already imports scantest needs nothing else.
//
// The policy itself lives in internal/testloader, which imports nothing: the scanner's own tests are
// in-package and this package imports the scanner, so a policy declared here could not be read from
// where most of the scans are.
const (
	LoaderSource   = testloader.LoaderSource
	LoaderCompiled = testloader.LoaderCompiled
	LoaderOwn      = testloader.LoaderOwn

	LoaderEnv = testloader.Env
)

// TestLoader reports which loader the suites run under. See [testloader.Selected].
func TestLoader() testloader.Loader { return testloader.Selected() }

// ApplyLoader writes the selected loader onto opts and returns it, for chaining onto a call.
//
// It is deliberately a WRITE rather than a default: a test that has an opinion about how its graph
// is loaded must not route through here at all, because there is no way to tell a field left at its
// zero value from one set to it on purpose. Those tests set their options directly, and say so.
//
// Options.FS is the one thing it will not fight: a virtual filesystem forces the toolchain-free
// loader whatever anyone asks for, so asking for something else here would only produce a
// diagnostic about an intent that could not be met.
func ApplyLoader(opts *scanner.Options) *scanner.Options {
	return applyLoader(opts)
}

// applyLoader is the shared body, so packages that cannot import scantest can hold a copy of the
// three lines rather than of the reasoning.
func applyLoader(opts *scanner.Options) *scanner.Options {
	if opts == nil || opts.FS != nil {
		return opts
	}

	switch selected := testloader.Selected(); selected {
	case testloader.LoaderCompiled:
		opts.CompiledDependencies = true
	case testloader.LoaderOwn:
		opts.ToolchainFreeLoader = true
	case testloader.LoaderSource:
		// The shipped default: nothing to set.
	default:
		panic("scantest: no rule for loader " + string(selected))
	}

	return opts
}
