// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2228

// swagger:route GET /alpha alpha withSummary
//
// This becomes the summary.
//
// responses:
//
//	200: description: ok
func WithSummary() {}

// swagger:route GET /beta beta noSummary
//
// responses:
//
//	200: description: ok
func NoSummary() {}
