// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2652

// swagger:model LabelSelector
type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels"`
}

// swagger:model Selector
type Selector struct {
	// the name of Selector
	// required: true
	// example: apple selector
	Name string `json:"name"`

	// the condition of Selector
	// required: true
	// example: {"matchLabels":{}}
	PodSelector *LabelSelector `json:"podSelector"`
}
