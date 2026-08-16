// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2802

// swagger:model WrappedRequest
type WrappedRequest[T any] struct {
	Body struct {
		Body T `json:",inline"`
	}
}

// swagger:model Request
type Request struct {
	IntVal int `json:"intVal"`
	StrVal int `json:"strVal"`
}

// swagger:model WrappedRequestInstance
type WrappedRequestInstance WrappedRequest[Request]
