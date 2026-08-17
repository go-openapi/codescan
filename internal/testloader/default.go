// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !testloader_compiled && !testloader_own

package testloader

// defaultLoader is what the suites run under when nothing selects otherwise.
//
// The shipped default, so an ordinary `go test ./...` exercises the configuration a caller gets.
// Anything else has to be asked for, by the environment or by a build tag.
const defaultLoader = LoaderSource
