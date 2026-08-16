// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2119

// Widget probes SkipExtensions suppressing x-go-name / x-go-package (#2119).
//
// swagger:model
type Widget struct {
	Colour string `json:"colour"`
}
