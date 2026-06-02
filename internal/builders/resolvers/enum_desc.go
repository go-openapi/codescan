// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package resolvers

import oaispec "github.com/go-openapi/spec"

// ExtEnumDesc is the vendor-extension key the scanner uses to expose
// the per-enum-value documentation lines built from `swagger:enum`
// + const-block comments. It's a go-swagger concept (not part of
// the OpenAPI spec); the key lives in `resolvers` so every builder
// (schema, parameters, responses) reads and writes it through the
// same constant.
const ExtEnumDesc = "x-go-enum-desc"

// GetEnumDesc reads the x-go-enum-desc extension off a Swagger
// extensions map. Empty when absent.
//
// Consumers typically check this after a build pass to know whether
// they should append the per-value docs to a Description (parameters
// + response headers do this for the parameter/header
// description; the schema builder folds it in differently — see
// `handlers/dispatch_schema.go:clearStaleEnumDesc` for the
// override-cleanup dance).
func GetEnumDesc(extensions oaispec.Extensions) string {
	desc, _ := extensions.GetString(ExtEnumDesc)
	return desc
}
