// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package deep is the name-identity pathological-depth corpus member (reserved
// for Stage 3). Two packages a/mongo and b/mongo each declare
// `swagger:model Book`. Both the leaf name ("Book") AND the one-segment concat
// ("MongoBook") collide, so the concat rung cannot disambiguate — exercising the
// hierarchical / deeper-prefix fail-safe (AMongoBook / BMongoBook, or nested
// form). Today they merge.
package deep

import (
	amongo "github.com/go-openapi/codescan/fixtures/enhancements/name-identity-deep/a/mongo"
	bmongo "github.com/go-openapi/codescan/fixtures/enhancements/name-identity-deep/b/mongo"
)

// swagger:model Library
type Library struct {
	A amongo.Book `json:"a"`
	B bmongo.Book `json:"b"`
}
