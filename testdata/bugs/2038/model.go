// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2038

// Inner is embedded by the models below.
//
// swagger:model
type Inner struct {
	InnerField string `json:"inner_field"`
}

// Tagged embeds Inner WITH an explicit json tag. Go's encoding/json NESTS the
// embedded value under "inner" (it is no longer promoted). The generated spec
// must match: an `inner` property, not promoted inner_field (go-swagger#2038).
//
// swagger:model
type Tagged struct {
	OuterField string `json:"outer_field"`
	Inner      `json:"inner"`
}

// Untagged embeds Inner WITHOUT a json tag — Go promotes (flattens) its fields.
// This is the green guard rail: promotion is correct here.
//
// swagger:model
type Untagged struct {
	OuterField string `json:"outer_field"`
	Inner
}

// Squashed embeds Inner but squashes it with `json:"-"`. Go's encoding/json
// drops the embed entirely: it is neither promoted nor nested. The schema must
// contain only outer_field (go-swagger#2038).
//
// swagger:model
type Squashed struct {
	OuterField string `json:"outer_field"`
	Inner      `json:"-"`
}

// TaggedAllOf embeds Inner with BOTH an explicit json tag AND a swagger:allOf
// annotation. The explicit composition annotation wins: the embed becomes an
// allOf member ($ref to Inner), the json tag does NOT cause nesting. This proves
// the json-tag nesting path fires only for plain (non-allOf) embeds.
//
// swagger:model
type TaggedAllOf struct {
	OuterField string `json:"outer_field"`
	// swagger:allOf
	Inner `json:"inner"`
}
