// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3013 reproduces go-swagger issue #3013 ("How to set an example
// value for array/string response type?"): an `example:` annotation on a
// response whose body is a top-level array type is dropped.
package bug3013

// Ntp servers
//
// swagger:response getNtpServersResponse
// example: ["10.10.10.10","20.20.20.20"]
type getNtpServersResponse []string

// swagger:route GET /ntp-servers ntp getNtpServers
//
// responses:
//
//	200: getNtpServersResponse
func getNtpServersHandler() {}
