// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/types"

	"golang.org/x/tools/go/packages"
)

// PkgForPath exposes the internal package lookup for test use only.
//
// It is not part of the production API; production code resolves packages through typed entry
// points like FindDecl/FindModel/DeclForType.
func (s *ScanCtx) PkgForPath(pkgPath string) (*packages.Package, bool) {
	v, ok := s.app.AllPackages[pkgPath]
	return v, ok
}

// PackagesRead reports how many of the loaded packages carry syntax, against how many were loaded.
//
// Test-only, and the only way to see the read-back policy from outside: a package the load took types-only has no
// syntax until something asks it for a declaration, so the first number is what the scan has actually paid to parse.
func (s *ScanCtx) PackagesRead() (read, loaded int) {
	for _, pkg := range s.app.AllPackages {
		if len(pkg.Syntax) > 0 {
			read++
		}
	}

	return read, len(s.app.AllPackages)
}

// DropExpressionTypes strips every loaded package's record of what its type expressions denote,
// keeping the record of what its declarations define.
//
// Test-only. It reproduces the shape a package read from compiled export data with its source
// parsed alongside is in: the package scope is complete and its declarations are joined to it by
// name, while types.Info.Types — which no bridging can rebuild, since the field distinguishing a
// type from a value is unexported — stays empty.
func (s *ScanCtx) DropExpressionTypes() {
	for _, pkg := range s.app.AllPackages {
		if pkg.TypesInfo == nil {
			continue
		}
		pkg.TypesInfo = &types.Info{Defs: pkg.TypesInfo.Defs}
	}
}
