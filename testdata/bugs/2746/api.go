// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2746

// UUID is a string formatted as a uuid.
//
// swagger:strfmt uuid
type UUID string

// swagger:model FilterBy
type FilterBy struct {
	VehicleTypes []UUID `json:"vehicleTypes"`
}
