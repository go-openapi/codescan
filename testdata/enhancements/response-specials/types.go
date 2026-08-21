// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_specials witnesses a `swagger:response` declared directly on a recognized stdlib
// type, in both spellings a Go author can reach it by.
//
// A response body that IS a timestamp, an opaque stream or an arbitrary-precision number is
// legitimate, and the two spellings must agree. They did not: `type Stamp time.Time` was caught by
// the written-RHS redirect and rendered `{string, date-time}`, while `type Stamp = time.Time`
// arrived AS time.Time and had nothing to catch it. time.Time is a STRUCT underneath, so dispatching
// on the underlying shape read it as a response struct whose fields become headers — it exports none,
// so the response carried a description and no schema at all.
//
// The recognizers answer from the object alone, so they belong ahead of the shape dispatch rather
// than inside one of its arms. `any` and `error` still refuse: they are not responses, and rendering
// them would be worse than saying so.
//
// See [§special-types](../../../internal/builders/schema/README.md#special-types).
package response_specials

import (
	"encoding/json"
	"io"
	"math/big"
	"time"
)

// Defined-type spelling — reaches the recognizers through the written right-hand side.

// swagger:response definedStamp
type DefinedStamp time.Time

// swagger:response definedRaw
type DefinedRaw json.RawMessage

// swagger:response definedStream
type DefinedStream io.Reader

// swagger:response definedWhole
type DefinedWhole big.Int

// Alias spelling — arrives as the stdlib type itself, and used to fall through to the struct arm.

// swagger:response aliasedStamp
type AliasedStamp = time.Time

// swagger:response aliasedRaw
type AliasedRaw = json.RawMessage

// swagger:response aliasedStream
type AliasedStream = io.Reader

// swagger:response aliasedWhole
type AliasedWhole = big.Int

// swagger:response aliasedFraction
type AliasedFraction = big.Rat

// Local-alias spelling — the declaration is written over an alias, which is what go1.27 made of
// json.RawMessage: `type DefinedRaw json.RawMessage` now has an ALIAS on its right-hand side. The
// redirect that carries the named layer to the recognizers has to follow that too.

// Stamped is a local alias of time.Time.
type Stamped = time.Time

// Rawish is a local alias of json.RawMessage.
type Rawish = json.RawMessage

// swagger:response viaAliasStamp
type ViaAliasStamp Stamped

// swagger:response viaAliasRaw
type ViaAliasRaw Rawish
