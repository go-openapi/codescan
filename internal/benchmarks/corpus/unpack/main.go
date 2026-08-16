// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command unpack makes the benchmark corpora available on disk and prints where
// they are, one `name<TAB>directory` line per corpus.
//
// It exists so run.sh does not have to grow its own copy of the layout: where
// the archive is, where it unpacks to and when it is stale are decided once, in
// the corpus package the Go harness uses.
//
//	go run ./internal/benchmarks/corpus/unpack
package main

import (
	"fmt"
	"os"

	"github.com/go-openapi/codescan/internal/benchmarks/corpus"
)

func main() {
	corpora, err := corpus.Ensure()
	if err != nil {
		fmt.Fprintln(os.Stderr, "unpack:", err)
		os.Exit(1)
	}

	for _, c := range corpora {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", c.Name, c.Dir)
	}
}
