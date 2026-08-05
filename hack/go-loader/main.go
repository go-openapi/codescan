// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command go-loader keeps codescan's toolchain-free package loader honest against the go command it
// stands in for.
//
// It does two jobs, both of which need a checkout of the Go distribution:
//
//   - sync: report whether the declarations copied out of the Go tree still match their originals,
//     and optionally refresh them;
//   - corpus: run the go command's own script-test trees through both loading strategies and compare
//     what each says the tree contains.
//
// Neither runs as part of the test suite, because both need a Go checkout that CI does not have. What
// is committed instead is the ledger the corpus run writes, so a future run against a newer Go shows
// its differences as a diff.
//
// Usage:
//
//	go run ./hack/go-loader -go /path/to/golang/go sync
//	go run ./hack/go-loader -go /path/to/golang/go sync -update
//	go run ./hack/go-loader -go /path/to/golang/go corpus
//	go run ./hack/go-loader -go /path/to/golang/go corpus -write-ledger
//	go run ./hack/go-loader -go /path/to/golang/go corpus -only list_ -v
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// errUsage reports a command line the tool cannot act on.
var errUsage = errors.New("usage")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-loader:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("go-loader", flag.ContinueOnError)
	goRoot := fs.String("go", "", "path to a checkout of the Go distribution (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *goRoot == "" {
		return fmt.Errorf("%w: -go is required; pass a checkout of github.com/golang/go", errUsage)
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("%w: expected a subcommand: sync or corpus", errUsage)
	}

	switch rest[0] {
	case "sync":
		return runSync(*goRoot, rest[1:])
	case "corpus":
		return runCorpus(*goRoot, rest[1:])
	default:
		return fmt.Errorf("%w: unknown subcommand %q", errUsage, rest[0])
	}
}
