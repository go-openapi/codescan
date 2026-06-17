// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1079

// Annotated is the only swagger:model in this package.
//
// swagger:model
type Annotated struct {
	Ref Referenced `json:"ref"`
}

// Referenced is NOT annotated but is referenced by Annotated.Ref.
type Referenced struct {
	X int `json:"x"`
}

// Unreferenced is NOT annotated and NOT referenced by anything.
type Unreferenced struct {
	Y int `json:"y"`
}
