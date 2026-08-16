// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug1026 probes whether a custom vendor extension (x-logo) can be
// declared in the swagger:meta block — the #1026 "x-logo feature" request is
// satisfied by generic InfoExtensions support rather than a dedicated knob.
//
//	Schemes: https
//	Host: localhost
//	Version: 1.0.0
//
//	InfoExtensions:
//	x-logo:
//	  url: ./images/logo.png
//	  backgroundColor: "#FFFFFF"
//
// swagger:meta
package bug1026
