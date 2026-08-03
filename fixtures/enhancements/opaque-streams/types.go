// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package opaque_streams witnesses the stdlib stream types across every position that can carry
// one.
//
// None of them had a recognizer, so they fell through to ordinary structural drilling — and an
// interface is the shape drilling handles worst. `io.Reader` as a formData parameter emitted no
// `type` at all (invalid: SimpleSchema requires one), and as a model field it published `io`'s own
// interfaces as definitions carrying io's godoc, inventing a `close` property of type string out
// of `Close() error`.
//
// A stream is opaque by construction: nothing in the declaration says what the bytes are. The
// recognizers therefore do not try to guess — they say "opaque bytes" in whichever way the
// position allows, and the author's `swagger:file` / `swagger:type` override still wins.
//
// See [§opaque-streams](../../../internal/builders/schema/README.md#opaque-streams).
package opaque_streams

import (
	"io"
	"mime/multipart"

	"github.com/go-openapi/runtime"
)

// StreamModel carries every recognized stream type as a model field.
//
// swagger:model StreamModel
type StreamModel struct {
	Reader         io.Reader               `json:"reader"`
	ReadCloser     io.ReadCloser           `json:"readCloser"`
	ReadSeeker     io.ReadSeeker           `json:"readSeeker"`
	ReadSeekCloser io.ReadSeekCloser       `json:"readSeekCloser"`
	ReadWriter     io.ReadWriter           `json:"readWriter"`
	ReaderAt       io.ReaderAt             `json:"readerAt"`
	ReaderFrom     io.ReaderFrom           `json:"readerFrom"`
	LimitedReader  io.LimitedReader        `json:"limitedReader"`
	ByteReader     io.ByteReader           `json:"byteReader"`
	ByteScanner    io.ByteScanner          `json:"byteScanner"`
	Named          runtime.NamedReadCloser `json:"named"`
	MultipartFile  multipart.File          `json:"multipartFile"`
}

// WriterModel holds the type deliberately left OUT of the recognized set.
//
// An API payload does not plausibly contain an `io.Writer` — a sink the caller writes into is not
// something that goes on the wire. Recognizing it would be inventing an intent, so it keeps
// whatever the structural walk makes of it and the author overrides if they meant something.
//
// swagger:model WriterModel
type WriterModel struct {
	Writer io.Writer `json:"writer"`
}

// OverriddenModel is the control: an explicit annotation still wins over the recognizer.
//
// swagger:model OverriddenModel
type OverriddenModel struct {
	// Blob says what the bytes are, so the recognizer must not overrule it.
	//
	// swagger:strfmt base64
	Blob io.Reader `json:"blob"`
}

// UploadParams reaches the stream types from each parameter location.
//
// swagger:parameters uploadStream
type UploadParams struct {
	// Upload is the canonical file-upload shape.
	//
	// in: formData
	Upload io.Reader `json:"upload"`

	// Doc is the same shape spelled with multipart.File.
	//
	// in: formData
	Doc multipart.File `json:"doc"`

	// Body carries the stream as the request body.
	//
	// in: body
	Body io.ReadCloser `json:"body"`

	// Marker is a stream in a location where it makes little sense, kept so the witness records
	// what such a declaration produces rather than leaving it undefined.
	//
	// in: query
	Marker io.Reader `json:"marker"`
}

// StreamResponse returns a stream as the response body.
//
// swagger:response streamResponse
type StreamResponse struct {
	// in: body
	Body io.ReadCloser

	// XChecksum is a stream reached through a response header.
	//
	// in: header
	XChecksum io.Reader
}
