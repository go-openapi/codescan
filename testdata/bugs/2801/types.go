// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2801

type WrappedRequest[T any] struct {
	// in: body
	Body struct {
		Body T `json:",inline"`
	}
}

// swagger:model Request
type Request struct {
	Val int `json:"int"`
}
