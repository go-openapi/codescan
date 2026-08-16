// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestWASIArtifactMatchesNativeScan builds cmd/genspec-wasi for wasip1/wasm and runs it under a WASI
// runtime, with the fixture tree mounted into the guest.
//
// The guest has no process model, so nothing it does can reach the go command: whatever spec comes
// back was produced by parsing and type-checking alone. Comparing it against an in-process scan is
// what makes that a claim about correctness rather than about not crashing.
func TestWASIArtifactMatchesNativeScan(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm artifact and runs it under an interpreter; minutes, not seconds")
	}
	t.Parallel()

	runtimeName, runtimeArgs := findWASIRuntime(t)
	artifact := buildWASIArtifact(t)

	const pattern = "./goparsing/petstore/..."
	fixtures, err := filepath.Abs("../../testdata")
	require.NoError(t, err)

	// The guest resolves the standard library and the module cache by path, so it has to be told
	// where they are: nothing in a WASI environment can ask the go command.
	// Concat rather than append: runtimeArgs comes from a helper and appending onto it would write
	// through whatever backing array that helper handed out.
	args := slices.Concat(runtimeArgs, []string{
		artifact, "-quiet", "-loader=own", "-workdir", fixtures,
		"-goos", "linux", "-goarch", "amd64", pattern,
	})

	cmd := exec.CommandContext(t.Context(), runtimeName, args...)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	require.NoError(t, err, "running the artifact under %s failed: %s", runtimeName, stderr.String())

	var fromGuest map[string]any
	require.NoError(t, json.Unmarshal(out, &fromGuest))

	// The control: the same scan, in this process, through the stock loader.
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{pattern},
		WorkDir:    fixtures,
		ScanModels: true,
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	require.NoError(t, err)

	native, err := json.Marshal(doc)
	require.NoError(t, err)
	var fromHost map[string]any
	require.NoError(t, json.Unmarshal(native, &fromHost))

	assert.Equal(t, fromHost, fromGuest,
		"the specification produced without a toolchain differs from the one produced with it")
}

// findWASIRuntime returns a runtime able to execute a Go wasip1 binary, plus the arguments that mount
// the host filesystem and forward the environment. Mount syntax differs per runtime.
func findWASIRuntime(t *testing.T) (string, []string) {
	t.Helper()

	forward := []string{"GOROOT", "GOMODCACHE", "GOPATH", "HOME"}

	if path, err := exec.LookPath("wazero"); err == nil {
		args := []string{"run", "-mount=/:/"}
		for _, k := range forward {
			if v := os.Getenv(k); v != "" {
				args = append(args, "-env", k+"="+v)
			}
		}

		return path, args
	}

	if path, err := exec.LookPath("wasmtime"); err == nil {
		args := []string{"run", "--dir", "/::/"}
		for _, k := range forward {
			if v := os.Getenv(k); v != "" {
				args = append(args, "--env", k+"="+v)
			}
		}

		return path, append(args, "--")
	}

	t.Skip("no WASI runtime on PATH (install wazero or wasmtime)")

	return "", nil
}

// buildWASIArtifact cross-compiles cmd/genspec-wasi to wasip1/wasm and returns its path.
func buildWASIArtifact(t *testing.T, tags ...string) string {
	t.Helper()

	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go command available to build the artifact")
	}

	artifact := filepath.Join(t.TempDir(), "genspec-wasi.wasm")
	buildArgs := []string{"build"}
	if len(tags) > 0 {
		buildArgs = append(buildArgs, "-tags", strings.Join(tags, ","))
	}
	buildArgs = append(buildArgs, "-o", artifact, "github.com/go-openapi/codescan/cmd/genspec-wasi")

	cmd := exec.CommandContext(t.Context(), goBin, buildArgs...)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "cross-compiling the artifact failed: %s", out)

	return artifact
}

// TestWASIArtifactIsSelfContained runs a build that carries the standard library's export data
// inside it, with nothing mounted but the project.
//
// This is the shape a browser needs: no GOROOT to ship, no toolchain, and no host filesystem beyond
// the sources the user supplied.
func TestWASIArtifactIsSelfContained(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a wasm artifact and runs it under an interpreter")
	}
	t.Parallel()

	// The embedded archive is generated, not committed: without it there is nothing to test.
	if _, err := os.Stat(filepath.Join("..", "exportdata", "exportdata.zip")); err != nil {
		t.Skip("no embedded export data (run: go run ./hack/genexportdata -out internal/exportdata/exportdata.zip std)")
	}

	runtimeName, _ := findWASIRuntime(t)
	artifact := buildWASIArtifact(t, "exportdata")

	fixtures, err := filepath.Abs("../../testdata")
	require.NoError(t, err)

	// Only the fixture tree is mounted, and no GOROOT is named. Anything the scan needs from the
	// standard library has to come from inside the binary.
	const pattern = "./enhancements/named-basic/..."
	args := slices.Concat(mountOnly(runtimeName, fixtures), []string{
		artifact, "-quiet", "-loader=own", "-workdir", fixtures,
		"-goos", "linux", "-goarch", "amd64", pattern,
	})

	cmd := exec.CommandContext(t.Context(), runtimeName, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	require.NoError(t, err, "self-contained run failed: %s", stderr.String())

	var fromGuest map[string]any
	require.NoError(t, json.Unmarshal(out, &fromGuest))

	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{pattern},
		WorkDir:    fixtures,
		ScanModels: true,
		GOOS:       "linux",
		GOARCH:     "amd64",
	})
	require.NoError(t, err)

	native, err := json.Marshal(doc)
	require.NoError(t, err)
	var fromHost map[string]any
	require.NoError(t, json.Unmarshal(native, &fromHost))

	assert.Equal(t, fromHost, fromGuest,
		"a self-contained scan differs from one with the whole toolchain available")
}

// mountOnly builds the runtime arguments that expose exactly one directory and nothing else.
func mountOnly(runtimeName, dir string) []string {
	if strings.Contains(runtimeName, "wasmtime") {
		return []string{"run", "--dir", dir + "::" + dir, "--"}
	}

	return []string{"run", "-mount=" + dir + ":" + dir}
}
