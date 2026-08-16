// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_space_body_quirk locks the legacy parser's
// interpretation of `200: body Pet ...` (space-separated, NOT
// colon-attached `body:Pet`). Legacy parseTags treats the first
// untagged token as a response name — so "body" becomes the response
// ref and "Pet the pet" becomes the description.
//
// This is malformed input that the parser silently mishandles; the
// fixture locks the misinterpretation so any post-refactor change in
// behaviour shows up in the golden diff for explicit review.
package routes_responses_space_body_quirk

// GetPet swagger:route GET /pets/{id} pets getPetSpaceQuirk
//
// Get a pet.
//
// Responses:
//
//	200: body Pet the pet
func GetPet() {}
