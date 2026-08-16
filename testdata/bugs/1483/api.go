// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1483

// swagger:parameters snapVolumeGroup
type groupSnapCreateRequest struct {
	// in: body
	Body struct {
		ID     string            `json:"id"`
		Labels map[string]string `json:"labels"`
	}
}

// swagger:route POST /snap snap snapVolumeGroup
//
// Snap.
//
// responses:
//
//	200: description: ok
func SnapVolumeGroup() {}
