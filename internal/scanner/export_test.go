// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import "golang.org/x/tools/go/packages"

// PkgForPath exposes the internal package lookup for test use only.
// It is not part of the production API; production code resolves packages
// through typed entry points like FindDecl/FindModel/DeclForType.
func (s *ScanCtx) PkgForPath(pkgPath string) (*packages.Package, bool) {
	v, ok := s.app.AllPackages[pkgPath]
	return v, ok
}
