// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2687

// MyType description
//
// +genclient
// +kubebuilder:resource:shortName=mytype
// +kubebuilder:subresource:status
// swagger:model Application
type MyType struct {
	// Name of the type
	// required: true
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}
