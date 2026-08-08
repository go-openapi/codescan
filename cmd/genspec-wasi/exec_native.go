// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !wasm

package main

// canExec reports whether this build can start a subprocess.
func canExec() bool { return true }
