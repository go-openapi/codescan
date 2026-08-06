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
