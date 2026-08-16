// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2248

import "time"

// IndexedNodes is a response with a time.Duration header (#2248).
//
// swagger:response indexedNodes
type IndexedNodes struct {
	// last contact
	// in: header
	LastContact time.Duration `json:"lastContact"`
	// in: body
	Body string
}
