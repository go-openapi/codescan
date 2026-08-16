// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package unsupportedgotype carries a model with a field whose Go type
// cannot be represented in Swagger 2.0. The scanner drops the field and
// records a validate.unsupported-go-type warning (the diagnostic that
// replaced the legacy stderr "unsupported Go type" log line).
package unsupportedgotype

// Widget is a model with one representable field and one that codescan
// cannot translate.
//
// swagger:model Widget
type Widget struct {
	// Name is a normal field.
	Name string `json:"name"`

	// Weird is a complex number — Swagger 2.0 has no such type, so the
	// field is dropped with a validate.unsupported-go-type warning.
	Weird complex128 `json:"weird"`
}
