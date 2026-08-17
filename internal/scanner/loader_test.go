// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import "github.com/go-openapi/codescan/internal/testloader"

// withTestLoader writes the loader this run was asked for onto opts.
//
// The same policy scantest.ApplyLoader applies, reached directly because these tests are IN-package:
// scantest imports this package, so importing scantest back would be a cycle. internal/testloader
// imports nothing, which is what makes it reachable from both sides.
//
// Tests that pin a loader themselves do not call this - there is no telling a field left at its zero
// value from one set to it on purpose.
func withTestLoader(opts *Options) *Options {
	if opts == nil || opts.FS != nil {
		return opts
	}

	switch selected := testloader.Selected(); selected {
	case testloader.LoaderCompiled:
		opts.CompiledDependencies = true
	case testloader.LoaderOwn:
		opts.ToolchainFreeLoader = true
	case testloader.LoaderSource:
	default:
		panic("scanner: no rule for loader " + string(selected))
	}

	return opts
}
