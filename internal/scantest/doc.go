// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package scantest exposes utilities for testing the codescan packages.
//
// # Choosing the loader a test run uses
//
// The suites emit the same document however the package graph was fetched, so which loader runs is
// a question about what the run costs. [ApplyLoader] is the one place that decides, reading
// [LoaderEnv] or the build tag the binary was compiled with:
//
//	go test ./...                                       # the shipped default, source dependencies
//	CODESCAN_TEST_LOADER=compiled go test ./...          # dependency types from the build cache
//	CODESCAN_TEST_LOADER=own go test ./...               # codescan's own loader, no go command
//	go test -tags=testloader_compiled ./...              # the same, selected at build time
//
// The environment variable is for a person at a terminal. The build tags exist because CI reaches
// `go test` through a shared workflow that forwards flags and not the environment.
//
// It is worth having because the harness is the shape compiled dependencies were made for: several
// hundred scans of small trees over one dependency closure, so the closure is compiled once and
// every scan after the first reads it back. Over the whole repository that is a quarter of the CPU,
// and a runner with two cores spends most of that difference on the wall.
//
// Two kinds of test do not come through here. Those that pin a loader themselves — the agreement
// A/B, the virtual-filesystem and export-data suites — call codescan.Run directly, because there is
// no way to tell a field left at its zero value from one set to it on purpose. And the runnable
// examples under docs/ show a caller how to call the API, so they use the real defaults and nothing
// else.
package scantest
