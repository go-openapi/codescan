// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routebody hosts the legacy regex-era body parsers that
// consume the indented "parameters:", "responses:", and "extensions:"
// blocks inside `swagger:route` comment docs.
//
// These parsers are the last citadel of the pre-grammar pipeline.
// They are consumed exclusively by internal/builders/routes/bridge.go
// — no other builder touches them. When routes/bridge.go grows a
// grammar-native body pipeline, this whole package can be deleted.
package routebody

import "regexp"

// rxCommentPrefix matches leading comment noise (whitespace, tabs,
// slashes, asterisks, dashes, optional markdown table pipe) before
// a keyword. Mirrors parsers.rxCommentPrefix — duplicated here so
// this package is self-contained and doesn't re-import parsers/.
const rxCommentPrefix = `^[\p{Zs}\t/\*-]*\|?\p{Zs}*`

var (
	rxResponses  = regexp.MustCompile(rxCommentPrefix + `[Rr]esponses\p{Zs}*:`)
	rxParameters = regexp.MustCompile(rxCommentPrefix + `[Pp]arameters\p{Zs}*:`)
	rxExtensions = regexp.MustCompile(rxCommentPrefix + `[Ee]xtensions\p{Zs}*:`)

	rxAllowedExtensions = regexp.MustCompile(`^[Xx]-`)

	// rxUncommentHeaders strips leading comment-marker noise from a
	// raw line. Consumed by helpers.CleanupScannerLines in the
	// extensions body parser.
	rxUncommentHeaders = regexp.MustCompile(`^[\p{Zs}\t/\*-]*\|?`)
)
