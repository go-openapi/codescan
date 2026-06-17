// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1459

// Cfg shows that a map field accepts a custom `example` with meaningful keys
// (go-swagger#1459) — the Swagger UI's additionalProp1/2/3 placeholders only
// appear when no example is set.
//
// swagger:model
type Cfg struct {
	// settings
	// example: {"timeout":"30s","retries":"3"}
	Settings map[string]string `json:"settings"`
}
