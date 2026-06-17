// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2441

// swagger:parameters uploadRawB
type rawParamsB struct {
	// raw video bytes
	// in: body
	// swagger:strfmt binary
	Body string
}

// swagger:route POST /rawb rawb uploadRawB
//
// Upload raw b.
//
// responses:
//
//	200: description: ok
func UploadRawB() {}
