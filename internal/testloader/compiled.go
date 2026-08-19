// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build testloader_compiled

package testloader

// defaultLoader is the loader the suites run under when nothing selects otherwise.
//
// Selected by `-tags=testloader_compiled`, which is how a CI lane asks for it: the shared test
// workflow forwards flags to `go test` and does not forward the environment.
const defaultLoader = LoaderCompiled
