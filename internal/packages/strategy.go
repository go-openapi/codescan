// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"strconv"

	"golang.org/x/tools/go/packages"
)

// Strategy names a way of resolving a package graph.
//
// Both strategies answer the same question and, across codescan's fixture corpus, give the same
// answer. They differ in what they need to run, which is the only reason a caller picks one.
type Strategy int

const (
	// StrategyGoPackages delegates to golang.org/x/tools/go/packages, which resolves the graph by
	// running `go list`. Authoritative — it is the go command's own answer — at the price of needing an
	// installed toolchain and the ability to start a process.
	//
	// The zero value, so a Loader that is asked for nothing behaves as codescan always has.
	StrategyGoPackages Strategy = iota

	// StrategyToolchainFree resolves the graph in pure Go: patterns to directories, build-constraint
	// selection through go/build, parse, type-check. Needs no toolchain, no GOROOT beyond the source it
	// reads, and no exec — which is what makes it the only strategy available inside a WebAssembly
	// guest, or against a virtual filesystem.
	StrategyToolchainFree
)

// String renders a Strategy for diagnostics.
func (s Strategy) String() string {
	switch s {
	case StrategyGoPackages:
		return "go/packages"
	case StrategyToolchainFree:
		return "toolchain-free"
	default:
		return "strategy(" + strconv.Itoa(int(s)) + ")"
	}
}

// WithStrategy selects how the package graph is resolved.
//
// The default is [StrategyGoPackages]. [WithFS] overrides this to [StrategyToolchainFree]: `go list`
// runs against the real filesystem, so it could not honour a virtual one even if asked, which means
// there is no coherent configuration being overridden.
func WithStrategy(s Strategy) Option {
	return func(o *options) { o.strategy = s }
}

// strategy reports which strategy this Loader will actually use, resolving the [WithFS] override.
func (l *Loader) strategy() Strategy {
	if l.opts.fsys != nil {
		return StrategyToolchainFree
	}

	return l.opts.strategy
}

// loadMode is the shape codescan needs, and what both strategies are contracted to deliver: named
// packages with their files, their transitive imports, and full syntax plus type information.
//
// The toolchain-free strategy produces this unconditionally — it has no cheaper mode to offer — so it
// is only the go/packages strategy that has to be told.
const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo
