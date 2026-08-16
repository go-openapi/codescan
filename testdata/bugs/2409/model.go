// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2409

// MyType probes a TYPE-level Extensions block (#2409).
//
// Extensions:
// ---
// x-custom-type-ext: hello
// ---
//
// swagger:model
type MyType struct {
	ID string `json:"id"`
}
