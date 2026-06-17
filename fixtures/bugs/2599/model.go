// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2599

// UUID is a custom, non-strfmt type (mirrors gofrs/uuid.UUID: an array under
// the hood) that the reporter cannot refactor away.
type UUID [16]byte

// Action exercises a field-level swagger:type override: ID is a UUID (which
// would otherwise render as a $ref/array) but the reporter wants it as a
// bare string in the spec.
//
// swagger:model
type Action struct {
	// swagger:type string
	ID UUID `json:"id"`

	ResourceID string `json:"resourceId"`
}
