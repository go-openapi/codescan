// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package api holds a body type referenced from another package (go-swagger#2002).
package api

// FooBarResponse is a body payload type declared in a separate package.
type FooBarResponse struct {
	Result string `json:"result"`
}
