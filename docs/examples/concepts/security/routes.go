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
// oauth2 with the read and write scopes. The Security: block is YAML: a sequence
// of requirement objects, scopes as a flow (or block) list.
//
// swagger:route POST /reports reports createReport
//
// Security:
//   - oauth2: [read, write]
//
// responses:
//   201: description: created

// archiveReport requires BOTH schemes at once — two keys in a single sequence
// item are ANDed into one requirement object (separate items would mean OR).
//
// swagger:route POST /reports/archive reports archiveReport
//
// Security:
//   - api_key: []
//     oauth2: [write]
//
// responses:
//   200: description: archived

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
