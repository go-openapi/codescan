// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1887

import "mime/multipart"

// swagger:parameters uploadFile
type uploadParams struct {
	// The file to upload.
	// in: formData
	// swagger:file
	Upfile *multipart.FileHeader `json:"upfile"`
}

// swagger:route POST /upload files uploadFile
//
// Uploads a file.
//
// Consumes:
//   - multipart/form-data
//
// Responses:
//
//	200: description: ok
func UploadFile() {}
