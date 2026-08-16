// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1267

// Binary is a file-download response (#1267): a binary body. The body field is
// declared with swagger:strfmt binary so the schema is {type:string,
// format:binary}; the operation sets produces: application/octet-stream.
//
// swagger:response binary
type Binary struct {
	// in: body
	// swagger:strfmt binary
	Payload string `json:"body"`
}

// swagger:route GET /excel files generateExcel
//
// Download excel.
//
// produces:
//   - application/octet-stream
//
// responses:
//
//	200: binary
func GenerateExcel() {}
