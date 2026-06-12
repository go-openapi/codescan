// SPDX-License-Identifier: Apache-2.0

// Package basic shows the smallest possible use of codescan: point it at an
// annotated package and get back a *spec.Swagger document.
package basic

import (
	"github.com/go-openapi/codescan"
	"github.com/go-openapi/spec"
)

// ScanPetstore scans the annotated petstore package and returns the generated
// Swagger 2.0 specification.
//
// codescan works at the AST / go/types level — it never compiles or executes
// the scanned code, it only reads the source and its annotation comments.
//
// workDir is the directory package patterns are resolved against (the module
// root of the package being scanned); patterns are relative `go list`-style
// patterns, e.g. "./petstore" or "./..." for a whole tree.
func ScanPetstore(workDir string) (*spec.Swagger, error) {
	// snippet:runScan
	opts := &codescan.Options{
		WorkDir:    workDir,                // module root to resolve patterns from
		Packages:   []string{"./petstore"}, // relative package pattern
		ScanModels: true,                   // also emit definitions for swagger:model types
	}

	doc, err := codescan.Run(opts)
	if err != nil {
		return nil, err
	}
	// endsnippet:runScan

	return doc, nil
}
