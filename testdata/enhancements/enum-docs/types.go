// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package enum_docs exercises scanCtx.findEnumValue description building
// on const blocks that carry per-value doc comments and multi-name value
// specs so both the docs loop and the names separator are walked.
package enum_docs

// Priority is an annotated enum string.
//
// swagger:enum Priority
type Priority string

const (
	// PriorityLow is a low-priority level.
	PriorityLow Priority = "low"

	// PriorityMed is a medium-priority level.
	PriorityMed Priority = "medium"

	// PriorityHigh is a high-priority level.
	PriorityHigh Priority = "high"
)

// Channel is an enum with a multi-name value spec so that findEnumValue
// walks the separator branch in its names loop.
//
// swagger:enum Channel
type Channel string

const (
	// ChannelEmail and ChannelSMS share a single spec.
	ChannelEmail, ChannelSMS Channel = "email", "sms"

	// ChannelPush is the push notification channel.
	ChannelPush Channel = "push"
)

// Notification holds both enums so that the scanner emits schemas with
// the enriched descriptions produced by findEnumValue.
//
// swagger:model Notification
type Notification struct {
	// required: true
	ID int64 `json:"id"`

	// The priority level.
	Priority Priority `json:"priority"`

	// The delivery channel.
	Channel Channel `json:"channel"`
}
