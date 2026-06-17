// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package nocollision is the name-identity G5 control: distinct models with
// distinct names across packages, no collision anywhere. Its golden MUST stay
// byte-identical through every stage of the engine — the bare-name common case
// proving the deconfliction introduces zero churn when there is nothing to
// deconflict.
package nocollision

import "github.com/go-openapi/codescan/fixtures/enhancements/name-identity-no-collision/sub"

// swagger:model Alpha
type Alpha struct {
	Name string   `json:"name"`
	Beta sub.Beta `json:"beta"`
}
