// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2909 reproduces go-swagger issue #2909 ("Regular cannot generate
// swagger automatically"): a route path with an inline regex segment
// (gorilla/chi style, e.g. {id:[0-9]+}) is silently dropped — the route
// regex's path alphabet has no `[`/`]`, so the swagger:route line fails to
// match and no path is emitted (and no diagnostic is raised).
package bug2909

// swagger:route GET /items/{id:[0-9]+} items getItem
//
// responses:
//
//	200: description: ok
func getItem() {}
