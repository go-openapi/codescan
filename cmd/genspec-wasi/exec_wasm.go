// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build wasm

package main

// canExec reports whether this build can start a subprocess.
//
// WebAssembly has no process model under either wasip1 or js, so `go list` - and therefore
// packages.Load - can never run here.
func canExec() bool { return false }
