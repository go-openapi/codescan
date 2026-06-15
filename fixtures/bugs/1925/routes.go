// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1925

// swagger:parameters doReport
type reportParams struct {
	// in: body
	Report []map[string]interface{} `json:"report"`
}

// swagger:route POST /report reports doReport
//
// Report.
//
// Responses:
//
//	200: description: ok
func DoReport() {}
