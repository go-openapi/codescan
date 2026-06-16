// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// The Documentation of project is described here.
// API Documentation Engine is Swagger official API-Dev, go-swagger.
//
// This is the OP repro from go-swagger#1395 (tab-indented swagger:meta with a
// Security requirement + SecurityDefinitions). The original "corrupted
// definitions" was a misclassified security entry; it now parses cleanly. (The
// OP wrote "SecurityDefinition" singular — a typo for the plural keyword used
// here.)
//
//				Schemes: http
//				Host: api.tipsyapp.net
//				BasePath: /lexy
//				Version: 1.0.0
//				Contact: Developer<dev@example.com>
//				Consumes:
//				- application/json
//				Produces:
//				- application/json
//				Security:
//				- APIKey
//				SecurityDefinitions:
//					APIKey:
//						type: apiKey
//						name: Authorization
//						in: header
// swagger:meta
package bug1395
