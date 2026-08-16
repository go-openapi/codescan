// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2251

// M probes map key types (go-swagger#2251). encoding/json marshals integer-kind
// keys as JSON string keys, so they ARE representable as additionalProperties —
// codescan currently drops them (only string / TextMarshaler keys are emitted).
//
// swagger:model
type M struct {
	// string key — the green guard rail (already emits additionalProperties).
	StrKey map[string]int `json:"strKey"`
	// int key — valid JSON, must emit additionalProperties (#2251).
	IntKey map[int]int `json:"intKey"`
	// named int64 key — likewise.
	I64Key map[int64]int `json:"i64Key"`
	// float key — encoding/json rejects it; must NOT emit additionalProperties
	// and must raise a diagnostic instead of dropping silently (§18 fail-loud).
	BadKey map[float64]int `json:"badKey"`
}
