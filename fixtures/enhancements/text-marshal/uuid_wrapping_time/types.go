// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package uuid_wrapping_time documents the asymmetry between *being*
// `time.Time` and *wrapping* it. A user type named `UUID` that embeds
// time.Time inherits MarshalText via method promotion (it does
// implement TextMarshaler), but the stdlib trio recognizer
// (`IsStdTime`) only matches the literal stdlib `time.Time` TypeName.
// Wrapping does not count.
//
// The UUID heuristic still matches by name though, so this type emits
// as `{string, uuid}` — not as a `date-time` string and not as a
// struct expansion.
package uuid_wrapping_time

import "time"

// UUID embeds time.Time and inherits its MarshalText method. The
// stdlib trio doesn't fire (UUID's Obj is in this package, not
// "time"), the user provided no `swagger:strfmt`, so the UUID name
// heuristic wins: emits as `{string, uuid}`.
type UUID struct {
	time.Time
}

// WrappingTimeContainer references UUID so the schema builder walks
// the heuristic path.
//
// swagger:model wrapping_time_container
type WrappingTimeContainer struct {
	// required: true
	ID UUID `json:"id"`
}
