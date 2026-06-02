// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_form_string exercises a form string parameter
// with allowempty: true.
package routes_params_form_string

// SubmitForm swagger:route POST /forms forms submitForm
//
// Submit a form.
//
// Parameters:
//   + name: comment
//     in: form
//     description: optional comment
//     type: string
//     allowempty: true
//
// Responses:
//
//	200: description: OK
func SubmitForm() {}
