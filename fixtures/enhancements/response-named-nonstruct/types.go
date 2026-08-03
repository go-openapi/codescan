// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_named_nonstruct witnesses a `swagger:response` declared on a
// NAMED type whose underlying is not a struct.
//
// That arm of the responses builder used to short-circuit on the stdlib time
// recognizer and on a local format helper, both of which wrote into a local
// schema and returned without attaching it — so the response came out carrying a
// description and nothing else. It also handed the sub-build the type's
// UNDERLYING rather than its declaration, which discards the named type the
// recognizer and the declaration's own classifiers key on.
//
// Each subject here is paired with the same type reached as a model field, whose
// rendering is pinned by its own witnesses. The two must agree: a response body
// and a model field are both full-schema positions, and there is no reason for a
// declaration to mean something different depending on which one reads it.
package response_named_nonstruct

import "time"

// Stamp is a named time.Time — the stdlib recognizer's subject.
type Stamp time.Time

// Emails is a named string slice carrying a non-special format, so the
// element-driven rule applies and the format belongs on the items.
//
// swagger:strfmt email
type Emails []string

// Code is a named string carrying a format.
//
// swagger:strfmt isbn
type Code string

// Count is a named integer with no annotation at all — the control that isolates
// "did the schema get attached" from "was it built correctly".
type Count int64

// Host reaches every subject as a MODEL FIELD, the control for the response side.
//
// swagger:model Host
type Host struct {
	// Stamp is the stdlib-recognizer subject.
	Stamp Stamp `json:"stamp"`

	// Emails is the element-driven-format subject.
	Emails Emails `json:"emails"`

	// Code is the whole-schema-format subject.
	Code Code `json:"code"`

	// Count is the unannotated control.
	Count Count `json:"count"`
}

// StampResp declares the response on the named time.Time.
//
// swagger:response stampResp
type StampResp Stamp

// EmailsResp declares the response on the formatted slice.
//
// swagger:response emailsResp
type EmailsResp = Emails

// CodeResp declares the response on the formatted string.
//
// swagger:response codeResp
type CodeResp = Code

// CountResp declares the response on the unannotated control.
//
// swagger:response countResp
type CountResp = Count

// HandlerStamp binds the stdlib-recognizer response.
//
// swagger:route GET /stamp resp opStamp
//
// Responses:
//
//	200: stampResp
func HandlerStamp() {}

// HandlerEmails binds the element-driven-format response.
//
// swagger:route GET /emails resp opEmails
//
// Responses:
//
//	200: emailsResp
func HandlerEmails() {}

// HandlerCode binds the whole-schema-format response.
//
// swagger:route GET /code resp opCode
//
// Responses:
//
//	200: codeResp
func HandlerCode() {}

// HandlerCount binds the control response.
//
// swagger:route GET /count resp opCount
//
// Responses:
//
//	200: countResp
func HandlerCount() {}
