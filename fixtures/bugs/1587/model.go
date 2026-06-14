// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1587

import "github.com/go-openapi/codescan/fixtures/bugs/1587/josev2"

// Client references a type from a package whose import-path tail (josev2)
// differs from its package name (jose). The legacy scanner failed with
// "no import found for jose".
//
// swagger:model oAuth2Client
type Client struct {
	JSONWebKeys *jose.JSONWebKeySet `json:"jwks,omitempty"`
}
