// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1512

// Sizes carries unsigned integer fields. Swagger 2.0 officially defines only
// int32 / int64 integer formats; go-swagger emits the Go-specific uint formats
// (uint64, uint32, …) as a vendor convention. For OAI-conformant / precision-safe
// output, a field can be overridden with `swagger:strfmt int64`, which yields the
// string-encoded {type: string, format: int64} representation.
//
// swagger:model
type Sizes struct {
	Big uint64 `json:"big"`

	Small uint32 `json:"small"`

	// swagger:strfmt int64
	Overridden uint64 `json:"overridden"`

	Signed int64 `json:"signed"`
}
