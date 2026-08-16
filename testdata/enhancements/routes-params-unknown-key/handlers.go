// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_unknown_key exercises silent-drop of unknown
// param keys: `colour: blue` is unknown to applyParamField and is
// stashed in extraData; processSchema only reads known schema keys
// (min/max/etc.) so `colour` is silently dropped.
package routes_params_unknown_key

// ListItems swagger:route GET /items items listItemsUnknown
//
// List items.
//
// Parameters:
//   + name: limit
//     in: query
//     description: max results
//     type: integer
//     colour: blue
//     min: 1
//
// Responses:
//
//	200: description: OK
func ListItems() {}
