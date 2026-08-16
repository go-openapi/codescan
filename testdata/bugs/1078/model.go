// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1078

import "time"

// Event carries a time.Time field, which JSON-serialises to a string and should
// be documented as {type: string, format: date-time}.
//
// swagger:model
type Event struct {
	CreatedAt time.Time `json:"createdAt"`

	Name string `json:"name"`
}
