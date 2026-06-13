// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package externaldocsobjects exercises externalDocs on non-meta OAIv2
// objects: operations (swagger:route + swagger:operation) and full schemas
// (swagger:model). On a simple-schema parameter (in != body) externalDocs is
// rejected with a diagnostic — it is a full-Schema-only keyword.
package externaldocsobjects

// swagger:route GET /route things routeOp
//
// responses:
//   200: description: ok
// externalDocs:
//   description: route docs
//   url: https://route.example.org
func routeOp() {}

// swagger:operation GET /operation things operationOp
//
// ---
// externalDocs:
//   description: operation docs
//   url: https://operation.example.org
// responses:
//   "200":
//     description: ok
func operationOp() {}

// Model carries externalDocs at the schema level.
//
// externalDocs:
//   description: model docs
//   url: https://model.example.org
//
// swagger:model Model
type Model struct {
	Name string `json:"name"`
}

// QueryParams has a simple (query) parameter wrongly carrying externalDocs;
// it is dropped with an unsupported-in-simple-schema diagnostic.
//
// swagger:parameters listThings
type QueryParams struct {
	// Filter query parameter.
	//
	// in: query
	// externalDocs:
	//   description: nope
	//   url: https://nope.example.org
	Filter string `json:"filter"`
}

// swagger:route GET /list things listThings
//
// responses:
//   200: description: ok
func listThings() {}
