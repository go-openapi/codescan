// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2296

// Embedded carries an anonymous struct field.
type Embedded struct {
	Inner struct {
		X string `json:"x"`
	} `json:"inner"`
}

// C embeds Embedded (embedded meets anonymous struct — #2296 panic).
//
// swagger:model
type C struct {
	Embedded
	Name string `json:"name"`
}
