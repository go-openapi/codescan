// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2985 reproduces go-swagger issue #2985 ("Need minAttributes and
// maxAttributes in the swagger:model annotation"): code-to-spec lacks
// minProperties / maxProperties support that spec-to-code already has.
package bug2985

// MyObjectType is a free-form object with property-count bounds.
//
// minProperties: 1
// maxProperties: 10
//
// swagger:model MyObjectType
type MyObjectType map[string]interface{}
