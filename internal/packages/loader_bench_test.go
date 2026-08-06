// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package packages_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/packages"
	"github.com/go-openapi/testify/v2/require"
)

// Loading is where a scan spends its time, and under WebAssembly — no threads, no `go list` to hand the work to —
// it is the only place it can spend it.
//
// These exist so that cost stays visible: allocation is the thing to watch, since the profile is dominated by the
// collector rather than by any computation of ours.
//
//	go test -run '^$' -bench Load -benchmem ./internal/packages/
func benchmarkLoad(b *testing.B, strategy packages.Strategy) {
	b.Helper()
	b.ReportAllocs()

	for b.Loop() {
		_, err := packages.NewLoader(packages.WithStrategy(strategy)).
			Load(&packages.Config{Dir: "../../fixtures/goparsing/petstore"}, "./...")
		require.NoError(b, err)
	}
}

func BenchmarkLoadToolchainFree(b *testing.B) {
	benchmarkLoad(b, packages.StrategyToolchainFree)
}

func BenchmarkLoadGoPackages(b *testing.B) {
	benchmarkLoad(b, packages.StrategyGoPackages)
}
