// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1542

// Metadata exercises example annotations on scalar and map fields (the #1542
// "examples in response schema" scenario).
//
// swagger:model
type Metadata struct {
	// Volume Name
	// example: OpenEBS Volume
	VolumeName string `json:"name"`

	// example: {"com.example1.com":"SomeString"}
	Annotations map[string]string `json:"annotations"`
}
