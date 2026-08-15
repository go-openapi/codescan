// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import "fmt"

// The package loader, as a choice of three rather than the boolean the library takes.
//
// ToolchainFreeLoader is a bool, but the useful third answer - "whichever one can run here" - is not
// expressible as one, and it is the answer a caller wants by default. So the flag states the choice
// and [resolveLoader] reduces it, which is also where the boolean is discharged from the coverage
// guard's point of view.
const (
	loaderFlag = "loader"

	loaderGo   = "go"   // packages.Load, which runs `go list`
	loaderOwn  = "own"  // codescan's own loader: no toolchain, no subprocess
	loaderAuto = "auto" // own where there is no exec, go otherwise

	loaderHelp = `package loader: "go" runs go list, "own" needs no toolchain, "auto" picks own where there is no exec`
)

// resolveLoader reports whether to use codescan's own loader.
//
// "auto" asks whether this build can start a subprocess at all: on wasm it cannot, so `go list` is
// not an option and the choice makes itself.
func resolveLoader(mode string) (bool, error) {
	switch mode {
	case loaderOwn:
		return true, nil
	case loaderGo:
		return false, nil
	case loaderAuto:
		return !canExec(), nil
	default:
		return false, fmt.Errorf("%w: -%s %q is not one of %s, %s, %s",
			ErrBadFlag, loaderFlag, mode, loaderGo, loaderOwn, loaderAuto)
	}
}
