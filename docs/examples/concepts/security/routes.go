// SPDX-License-Identifier: Apache-2.0

package security

// snippet:routes

// listReports inherits the document-wide default requirement (api_key) — no
// Security: keyword is needed.
//
// swagger:route GET /reports reports listReports
//
// responses:
//   200: description: the reports

// createReport overrides the default with its own Security: requirement —
// oauth2 with the read and write scopes.
//
// swagger:route POST /reports reports createReport
//
// Security:
//   oauth2: read, write
//
// responses:
//   201: description: created

// publicReport opts out of the document default entirely — an empty
// `Security: []` emits an explicit empty requirement, marking the operation
// public regardless of the document-wide default.
//
// swagger:route GET /reports/public reports publicReport
//
// Security: []
//
// responses:
//   200: description: the public reports

// endsnippet:routes
