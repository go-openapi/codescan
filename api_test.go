// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

// Public-API smoke suite. Fixture-heavy tests live in internal/integration.

var enableDebug bool //nolint:gochecknoglobals // test flag registered in init

func init() { //nolint:gochecknoinits // registers test flags before TestMain
	flag.BoolVar(&enableDebug, "enable-debug", false, "enable debug output in tests")
}

func TestMain(m *testing.M) {
	flag.Parse()

	if !enableDebug {
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.SetOutput(os.Stderr)
	}

	os.Exit(m.Run())
}

func TestApplication_DebugLogging(t *testing.T) {
	// Exercises the logger.DebugLogf code path with Debug: true.
	_, err := Run(&Options{
		Packages:   []string{"./goparsing/petstore/..."},
		WorkDir:    "fixtures",
		ScanModels: true,
		Debug:      true,
	})

	require.NoError(t, err)
}

func TestRun_InvalidWorkDir(t *testing.T) {
	// Exercises the Run() error path when package loading fails.
	_, err := Run(&Options{
		Packages: []string{"./..."},
		WorkDir:  "/nonexistent/directory",
	})

	require.Error(t, err)
}

func TestSetEnumDoesNotPanic(t *testing.T) {
	// Regression: ensure Run() does not panic on minimal source with an enum.
	dir := t.TempDir()

	src := `
	package failure

	// swagger:model Order
	type Order struct {
		State State ` + "`json:\"state\"`" + `
	}

	// State represents the state of an order.
	// enum: ["created","processed"]
	type State string
	`
	err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(src), 0o600)
	require.NoError(t, err)

	goMod := `
	module failure
	go 1.23`
	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600)
	require.NoError(t, err)

	_, err = Run(&Options{
		WorkDir:    dir,
		ScanModels: true,
	})

	require.NoError(t, err)
}
