// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug613

// Ulimit stands in for an external referenced struct (#613).
type Ulimit struct {
	Name string `json:"name"`
	Soft int64  `json:"soft"`
}

// Resources references an external-like struct via a slice-of-pointer field.
//
// swagger:model
type Resources struct {
	Ulimits []*Ulimit `json:"ulimits"`
}
