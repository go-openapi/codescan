// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build testloader_own

package testloader

// defaultLoader is the loader the suites run under when nothing selects otherwise.
//
// Selected by `-tags=testloader_own`, for a lane that exercises the toolchain-free loader over the
// whole corpus rather than over the agreement A/B alone.
const defaultLoader = LoaderOwn
