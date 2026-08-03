// SPDX-License-Identifier: Apache-2.0

// Package streams holds the annotated declarations for the "File uploads and byte
// streams" how-to. streams_test.go scans it and writes the goldens the guide
// renders.
package streams

import (
	"io"
	"mime/multipart"
)

// snippet:model

// Attachment carries opaque byte streams as model fields.
//
// A stream says nothing about its own framing, so codescan does not invent one:
// each field renders as `{string, format: byte}` — the base64-encoded string
// Swagger 2.0 uses for arbitrary bytes.
//
// swagger:model
type Attachment struct {
	// Content is the attachment payload.
	Content io.Reader `json:"content"`

	// Thumbnail is a closeable stream; the same answer applies.
	Thumbnail io.ReadCloser `json:"thumbnail"`

	// Checksum says what its bytes are, so the annotation wins over the default.
	//
	// swagger:strfmt base64
	Checksum io.Reader `json:"checksum"`
}

// endsnippet:model

// snippet:params

// UploadParams uploads a file and its metadata.
//
// swagger:parameters uploadAttachment
type UploadParams struct {
	// Upload is the file to store.
	//
	// in: formData
	Upload multipart.File `json:"upload"`

	// Caption describes the upload.
	//
	// in: formData
	Caption string `json:"caption"`
}

// endsnippet:params

// swagger:route POST /attachments attachments uploadAttachment
//
// Uploads an attachment.
//
// Consumes:
//   - multipart/form-data
//
// Responses:
//
//	200: attachmentResponse

// AttachmentResponse returns the stored attachment.
//
// swagger:response attachmentResponse
type AttachmentResponse struct {
	// in: body
	Body Attachment
}
