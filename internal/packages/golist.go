// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package packages

import (
	"os"

	"golang.org/x/tools/go/packages"
)

// loadViaGoPackages resolves the graph with golang.org/x/tools/go/packages, which shells out to
// `go list`.
func (l *Loader) loadViaGoPackages(cfg *Config, patterns ...string) ([]*Package, error) {
	// Copied rather than mutated: Config belongs to the caller, and Load is documented as repeatable.
	effective := *cfg
	effective.Tests = false
	if effective.Mode == 0 {
		effective.Mode = loadMode
		if l.opts.compiledDeps {
			effective.Mode = compiledDepsMode
		}
	}
	if env := l.pinnedEnv(cfg.Env); env != nil {
		effective.Env = env
	}

	return packages.Load(&effective, patterns...)
}

// pinnedEnv renders the loader's GoEnv as environment assignments over base, or nil when nothing is
// pinned.
//
// The go command reads GOOS, GOARCH, GOFLAGS, GOWORK and GOEXPERIMENT from the environment and
// nowhere else, so anything the caller pinned has to be pushed there. Leaving it inherited is what
// made a pinned target apply to one strategy and not the other.
//
// A nil base means the current environment, as it does for packages.Load; assignments are appended,
// and the last one wins, so a pinned value overrides whatever was inherited.
func (l *Loader) pinnedEnv(base []string) []string {
	pinned := l.opts.goEnv.assignments()
	if len(pinned) == 0 {
		return nil
	}

	if base == nil {
		base = os.Environ()
	}

	env := make([]string, len(base), len(base)+len(pinned))
	copy(env, base)

	return append(env, pinned...)
}
