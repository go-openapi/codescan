// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package meta_lists_flex_forms witnesses Property.AsList covering
// the meta (swagger:meta) annotation surface for list-shaped
// keywords. The meta dispatcher in builders/spec/walker.go reads
// schemes / consumes / produces via the same Property.AsList seam
// the routes dispatcher uses, so the accepted forms are identical.
//
// swagger:meta
//
// Title:
//
//	Lists Flex Forms (meta)
//
// Description:
//
//	Witnesses inline + multi-line list forms on the meta surface.
//
// Schemes:
//   - http
//   - https
//   - ws
//
// Consumes: application/json
//
// Produces:
//
//	application/json
//	application/xml
package meta_lists_flex_forms
