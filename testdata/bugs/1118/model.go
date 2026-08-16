// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1118

// WithPeriod is a thing.
// swagger:model
type WithPeriod struct {
	A int `json:"a"`
}

// WithoutPeriod is a thing
// swagger:model
type WithoutPeriod struct {
	B int `json:"b"`
}

// MultiLine is a thing.
//
// And here is a longer description spanning a separate paragraph.
//
// swagger:model
type MultiLine struct {
	C int `json:"c"`
}
