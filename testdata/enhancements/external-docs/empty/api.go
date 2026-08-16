// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package empty has an ExternalDocs block with no fields; it must NOT emit a
// useless `externalDocs: {}` (the OAS object requires url).
//
//	Version: 1.0.0
//	ExternalDocs:
//
// swagger:meta
package empty
