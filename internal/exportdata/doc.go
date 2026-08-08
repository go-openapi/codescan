// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package exportdata optionally carries dependencies' export data inside the binary.
//
// A scan that cannot reach GOROOT or a module cache - a WASI guest with only the project mounted, a
// browser - still needs its dependencies' types, and reading them from precomputed export data costs
// a fraction of type-checking them from source. Carrying that data in the binary is what makes such
// a build self-contained.
//
// It is opt-in, because it is several megabytes that most builds have no use for:
//
//	go run ./hack/genexportdata -out internal/exportdata/exportdata.zip std github.com/go-openapi/...
//	go build -tags exportdata ./cmd/genspec-wasi
//
// Without the tag, [Embedded] reports that nothing is embedded and the caller falls back to whatever
// the environment offers. The archive is valid only for the toolchain that generated it.
package exportdata
