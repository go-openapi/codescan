// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package annotation_noise witnesses a classifier annotation written in a
// position that never consults it.
//
// `swagger:strfmt` and `swagger:type` in an EMBEDDED field's own comment were
// parsed, validated and discarded without a word, while the same annotation on a
// regular field one line away is honoured. The scanner rejects an UNKNOWN
// annotation in that same comment (see the unknown-annotation fixture), so the
// author got validation feedback implying the annotation was meaningful and
// nothing saying it had been dropped.
package annotation_noise

// Target is the embedded type. Its own declaration is where a format belongs.
type Target struct {
	// Left is a plain property.
	Left string `json:"left"`
}

// Scalar is a named basic used as a regular field, where the annotations DO work.
type Scalar int

// IneffectiveOnAllOf annotates an allOf embed with classifiers the arm ignores.
//
// swagger:model IneffectiveOnAllOf
type IneffectiveOnAllOf struct {
	// swagger:allOf
	// swagger:strfmt uuid
	// swagger:type string
	Target

	// Note is the composing struct's own field.
	Note string `json:"note"`
}

// IneffectiveOnPlain annotates a PLAIN embed with the same classifiers.
//
// swagger:model IneffectiveOnPlain
type IneffectiveOnPlain struct {
	// swagger:strfmt uuid
	Target

	// Note is the embedding struct's own field.
	Note string `json:"note"`
}

// EffectiveOnNamedEmbed gives the embed a json name, which makes it a single
// named property rather than a promotion — so the classifier IS consulted,
// exactly as on a regular field. Reporting it as ineffective was a false alarm.
//
// swagger:model EffectiveOnNamedEmbed
type EffectiveOnNamedEmbed struct {
	// swagger:strfmt uuid
	Target `json:"nested"`

	// Note is the embedding struct's own field.
	Note string `json:"note"`
}

// EffectiveOnField is the control: the same annotations on regular fields, where
// both are honoured.
//
// swagger:model EffectiveOnField
type EffectiveOnField struct {
	// Fmt takes the format.
	//
	// swagger:strfmt uuid
	Fmt Scalar `json:"fmt"`

	// Typ takes the type override.
	//
	// swagger:type string
	Typ Scalar `json:"typ"`
}
