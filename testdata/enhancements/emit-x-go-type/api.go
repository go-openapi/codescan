// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package emitxgotype exercises the EmitXGoType option: every emitted
// definition should carry an x-go-type vendor extension recording its
// fully-qualified originating Go type, alongside the existing x-go-name /
// x-go-package traceability extensions.
package emitxgotype

// Widget is a plain model whose definition records its Go origin.
//
// swagger:model
type Widget struct {
	// Colour names the widget colour.
	Colour string `json:"colour"`
}
