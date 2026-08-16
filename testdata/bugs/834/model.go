// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug834

// Widget probes that Format/Pattern/Example/Type are applied as schema fields,
// not concatenated into the description (#834).
//
// swagger:model
type Widget struct {
	// the code
	// pattern: ^[A-Z]+$
	// max length: 10
	// example: ABC
	Code string `json:"code"`
}
