// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2872 reproduces go-swagger issue #2872 ("ExternalDocs are not
// generating the 2.0 spec on swagger:meta"): an ExternalDocs block in
// swagger:meta is not emitted (KwExternalDocs exists in the grammar but is not
// wired into the meta/info builder).
//
//	Version: 1.0.0
//	ExternalDocs:
//	  description: foo bar
//	  url: https://example.org
//
// swagger:meta
package bug2872
