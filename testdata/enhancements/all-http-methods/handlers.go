// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package all_http_methods declares one handler per HTTP verb so that
// setPathOperation walks the PATCH, HEAD and OPTIONS branches alongside
// the already-tested GET/POST/PUT/DELETE.
package all_http_methods

// GetItem swagger:route GET /items items getItem
//
// Get an item by id.
//
// Responses:
//
//	200: description: OK
func GetItem() {}

// PostItem swagger:route POST /items items postItem
//
// Create an item.
//
// Responses:
//
//	201: description: Created
func PostItem() {}

// PutItem swagger:route PUT /items items putItem
//
// Replace an item.
//
// Responses:
//
//	200: description: OK
func PutItem() {}

// PatchItem swagger:route PATCH /items items patchItem
//
// Apply a partial update.
//
// Responses:
//
//	200: description: OK
func PatchItem() {}

// DeleteItem swagger:route DELETE /items items deleteItem
//
// Delete an item.
//
// Responses:
//
//	204: description: No Content
func DeleteItem() {}

// HeadItem swagger:route HEAD /items items headItem
//
// Probe an item.
//
// Responses:
//
//	200: description: OK
func HeadItem() {}

// OptionsItem swagger:route OPTIONS /items items optionsItem
//
// Describe the supported HTTP methods.
//
// Responses:
//
//	200: description: OK
func OptionsItem() {}
