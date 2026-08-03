// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package default_example_typing witnesses how `default:` and `example:` values
// are TYPED, across every site that accepts them.
//
// The two keywords are deliberately paired everywhere: they share
// `validations.ParseDefault` and the same dispatch arms, so a design choice for
// one applies to the other, and any divergence between them is itself a defect.
//
// # The reported defect
//
// A value on a TYPE DECLARATION is coerced against an empty type. The decl
// comment block is dispatched by `applyDeclCommentBlock` before the Go type is
// resolved onto the schema, so `SchemaTypeOf(ps)` is "" and `ParseDefault` falls
// back to a string. The same keyword on a struct field, where the type is known,
// coerces correctly. Each decl cell below has a field-site control carrying the
// identical literal.
//
// # Default VALUE vs default RESPONSE
//
// These are unrelated mechanisms and must not be conflated:
//
//   - a default VALUE is the `default:` keyword, legal in the schema, parameter,
//     header and items contexts;
//   - a default RESPONSE is the `default` code head in a route's `Responses:`
//     body, which names a response — it is not a value at all.
//
// `default:` is NOT legal in a response block context (`KwDefault`'s context set
// omits `CtxResponse`), so the two cannot collide. The route at the bottom pins
// the response sense while the types above pin the value sense.
package default_example_typing

// DeclInt carries an integer value on the declaration.
//
// swagger:model DeclInt
// default: 8080
// example: 9090
type DeclInt int

// DeclNumber carries a floating-point value on the declaration.
//
// swagger:model DeclNumber
// default: 1.5
// example: 2.5
type DeclNumber float64

// DeclBool carries a boolean value on the declaration.
//
// swagger:model DeclBool
// default: false
// example: true
type DeclBool bool

// DeclString carries a string value on the declaration — the one case where a
// string fallback is indistinguishable from a correct coercion, so it is the
// control for the controls.
//
// swagger:model DeclString
// default: auto
// example: manual
type DeclString string

// DeclIntSlice carries a JSON array on the declaration.
//
// swagger:model DeclIntSlice
// default: [1,2,3]
// example: [4,5]
type DeclIntSlice []int

// DeclEnumInt carries an enum alongside a default on the declaration, so the
// enum members and the default can be compared for consistent typing.
//
// swagger:model DeclEnumInt
// enum: 1,2,3
// default: 2
type DeclEnumInt int

// DeclUncoercible carries values that cannot be read as the declared type. Each
// must be DROPPED with a warning rather than emitted at the wrong type — a
// document carrying `"notanumber"` on an integer schema is one no validator
// accepts, whereas a document missing a default is merely incomplete.
//
// The enum is partially bad: 1 and 3 survive, "two" is dropped. That narrows a
// closed set, which is a real change to the author's contract, so the warning
// names the member.
//
// swagger:model DeclUncoercible
// default: notanumber
// example: alsonotanumber
// enum: 1, two, 3
type DeclUncoercible int

// FieldUncoercible is the field-site counterpart. It always dropped the value —
// but silently, which is the half of the defect that was invisible.
//
// swagger:model FieldUncoercible
type FieldUncoercible struct {
	// Port has an uncoercible default and example.
	//
	// default: notanumber
	// example: alsonotanumber
	Port int `json:"port"`

	// Grade has a partially uncoercible enum.
	//
	// enum: 1, two, 3
	Grade int `json:"grade"`
}

// FieldControls carries the identical literals at FIELD sites, where the Go type
// is already resolved when the keyword walk runs. Every property here is the
// control for the like-named declaration above.
//
// swagger:model FieldControls
type FieldControls struct {
	// Port is the integer control.
	//
	// default: 8080
	// example: 9090
	Port int `json:"port"`

	// Ratio is the floating-point control.
	//
	// default: 1.5
	// example: 2.5
	Ratio float64 `json:"ratio"`

	// Flag is the boolean control.
	//
	// default: false
	// example: true
	Flag bool `json:"flag"`

	// Mode is the string control.
	//
	// default: auto
	// example: manual
	Mode string `json:"mode"`

	// Numbers is the JSON-array control.
	//
	// default: [1,2,3]
	// example: [4,5]
	Numbers []int `json:"numbers"`

	// Grade is the enum control.
	//
	// enum: 1,2,3
	// default: 2
	Grade int `json:"grade"`
}

// TypingParams carries the same literals in parameter positions — non-body
// (SimpleSchema) and body (full schema).
//
// swagger:parameters typingOp
type TypingParams struct {
	// QueryPort is a non-body parameter: SimpleSchema, no $ref allowed.
	//
	// in: query
	// default: 8080
	// example: 9090
	QueryPort int `json:"queryPort"`

	// QueryFlag is a non-body boolean parameter.
	//
	// in: query
	// default: false
	QueryFlag bool `json:"queryFlag"`

	// Body is a body parameter; its fields are full-schema properties.
	//
	// in: body
	Body struct {
		// Retries is a body-schema property.
		//
		// default: 3
		// example: 5
		Retries int `json:"retries"`
	} `json:"body"`
}

// TypingResponse carries the same literals on response HEADERS, which are
// SimpleSchema locations, and on a body property.
//
// Note there is no `default:` on the response block itself — that keyword is not
// legal in a response context, and a default RESPONSE is expressed by the route
// below instead.
//
// swagger:response typingResponse
type TypingResponse struct {
	// XRateLimit is a response header: SimpleSchema.
	//
	// in: header
	// default: 60
	// example: 120
	XRateLimit int `json:"X-Rate-Limit"`

	// Body is the response payload.
	//
	// in: body
	Body struct {
		// Retries is a response-body property.
		//
		// default: 3
		// example: 5
		Retries int `json:"retries"`
	} `json:"body"`
}

// ErrorResponse is the operation's default response — the OTHER sense of
// "default", carried by a response code rather than a value.
//
// swagger:response errorResponse
type ErrorResponse struct {
	// Body is the error payload.
	//
	// in: body
	Body struct {
		// Message describes the failure.
		Message string `json:"message"`
	} `json:"body"`
}

// TypingHandler binds the parameters and both responses to an operation. The
// `default:` code head names a RESPONSE; it must never be read as a value.
//
// swagger:route GET /typing typing typingOp
//
// Responses:
//
//	200: typingResponse
//	default: errorResponse
func TypingHandler() {}
