// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug91

// swagger:operation GET /search search doSearch
//
// Search, declaring the query param inline (no wrapper struct, #91).
//
// ---
// parameters:
//   - name: type
//     in: query
//     type: string
//     description: the tag type
// responses:
//   '200':
//     description: ok
func DoSearch() {}
