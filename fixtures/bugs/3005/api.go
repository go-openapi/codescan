// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3005 covers go-swagger issue #3005 ("additionalProperties are
// lost when generating spec from code"): a model wants named properties AND
// additionalProperties. The reporter carried the latter on a json:"-" map
// field, which is muted; the resolution is the explicit type-level marker
// `swagger:additionalProperties <spec>`, which restates the value type.
package bug3005

// TestAdditionalProperties has a named property plus free-form number values.
//
// swagger:model TestAdditionalProperties
// swagger:additionalProperties number
type TestAdditionalProperties struct {
	// field1
	Field1 string `json:"field1,omitempty"`

	// The map that motivated #3005 stays muted (json:"-"); the marker above
	// supplies the additionalProperties schema.
	TestAdditionalProperties map[string]float64 `json:"-"`
}
