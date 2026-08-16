// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2871

// Widget is shared across operations and carries its OWN model-level example.
//
// swagger:model
type Widget struct {
	// example: model-default
	Name string `json:"name"`
}

// swagger:operation GET /widgets/{id} widgets getWidget
//
// Get a widget.
//
// ---
// responses:
//   '200':
//     description: found
//     examples:
//       application/json:
//         name: alice-200
//   '404':
//     description: missing
//     examples:
//       application/json:
//         name: not-found-404
func GetWidget() {}

// swagger:operation POST /widgets widgets createWidget
//
// Create a widget (same model, different per-response examples).
//
// ---
// responses:
//   '201':
//     description: created
//     examples:
//       application/json:
//         name: created-201
//   '409':
//     description: conflict
//     examples:
//       application/json:
//         name: conflict-409
func CreateWidget() {}
