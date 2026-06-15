// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2520

import "strings"

// String is a custom-marshaled wrapper with only an unexported field (#2520).
type String struct {
	val string
}

func (f *String) UnmarshalJSON(b []byte) error { f.val = strings.Trim(string(b), "\""); return nil }
func (f String) MarshalJSON() ([]byte, error)  { return []byte(f.val), nil }

// Holder uses the custom-marshaled type as a field.
//
// swagger:model
type Holder struct {
	Name String `json:"name"`
}
