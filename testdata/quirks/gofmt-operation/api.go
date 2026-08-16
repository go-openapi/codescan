// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package gofmtoperation is the operations counterpart of quirk F7: a
// swagger:operation YAML body in its gofmt-CANONICAL shape. The user writes
// column-0 top-level keys (responses, x-*) with space-indented children; gofmt
// then renders each top-level key as a prose line (one leading space) and each
// indented value block as a code block (a leading TAB, inner nesting preserved
// as spaces after the tab), separated by blank "//" lines. YAML refuses tab
// indentation, so the body once collapsed (responses -> default:"", extension
// -> null). The dedent now expands leading tabs before stripping the common
// indent, so the gofmt form parses identically to the space-indented form.
package gofmtoperation

// GetAws is the witness operation.
//
// swagger:operation GET /aws aws getAws
//
// AWS-integrated operation.
//
// ---
// responses:
//
//	'200':
//		description: ok
//
// x-amazon-apigateway-integration:
//
//	httpMethod: GET
//	passthroughBehavior: when_no_match
//	responses:
//		default:
//			statusCode: "200"
//	type: http_proxy
//	uri: https://proxy-url.com
func GetAws() {}
