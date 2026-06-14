// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1115

// Thing stands in for some.Thing in the issue.
type Thing struct {
	Name string `json:"name"`
}

// FooResponse is a foo response (non-pointer alias — always worked).
//
// swagger:model fooResponse
type FooResponse Thing

// BarResponse is a bar response (alias to a POINTER — used to warn
// "Missing parser for a *ast.StarExpr" and be omitted).
//
// swagger:model barResponse
type BarResponse *Thing
