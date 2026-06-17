// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package jose lives in a directory ("josev2") whose name does NOT match the
// package name ("jose") — like gopkg.in/square/go-jose.v2 in the issue.
package jose

// JSONWebKeySet is the referenced type.
type JSONWebKeySet struct {
	Keys []string `json:"keys"`
}
