// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2592

// swagger:model
type User struct {
	Name string `json:"name"`
}

// swagger:model
type Token struct {
	Token string `json:"token"`
}

// Combined is a named body type composing User + Token via swagger:allOf.
//
// swagger:model
type Combined struct {
	// swagger:allOf
	User
	// swagger:allOf
	Token
}

// swagger:response combinedResponse
type combinedResponse struct {
	// in: body
	Body Combined
}

// inline body with swagger:allOf embeds
//
// swagger:response inlineResponse
type inlineResponse struct {
	// in: body
	Body struct {
		// swagger:allOf
		User
		// swagger:allOf
		Token
	}
}
