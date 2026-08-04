// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/types"
	"strings"

	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/ifaces"
)

// buildFromTextMarshal renders a TextMarshaler-implementing type as a string.
//
// Six-step pipeline; user-annotation first, implicit recognizers next, generic fallback last.
//
// # Details
//
// See [§textmarshal-order](./README.md#textmarshal-order).
func (s *Builder) buildFromTextMarshal(tpe types.Type, target ifaces.SwaggerTypable) error {
	if typePtr, ok := tpe.(*types.Pointer); ok {
		return s.buildFromTextMarshal(typePtr.Elem(), target)
	}
	// Aliases route through buildAlias so RefAliases / TransparentAliases stay in charge of the alias
	// indirection.
	if typeAlias, ok := tpe.(*types.Alias); ok {
		return s.buildAlias(typeAlias, target)
	}

	typeNamed, ok := tpe.(*types.Named)
	if !ok {
		target.Typed("string", "")
		return nil
	}
	tio := typeNamed.Obj()

	// Explicit user annotation wins over implicit recognizers.
	if s.classifierTextMarshal(typeNamed, target) {
		return nil
	}
	// Implicit recognizers in priority order: certain (identity) before guessed (fuzzy name).
	if applySpecialType(tio, target, s.skipExtensions,
		recognizeError, recognizeTime, recognizeRawMessage, recognizeStdUUID, recognizeUUID) {
		return nil
	}
	// Generic fallback: x-go-type carries pkg.Name, so PkgForType-miss must bail (can't produce the
	// extension without the package).
	if _, found := s.Ctx.PkgForType(tpe); !found {
		return nil
	}
	target.Typed("string", "")
	target.AddExtension("x-go-type", tio.Pkg().Path()+"."+tio.Name())
	return nil
}

type recognizeType uint8

const (
	recognizedNone recognizeType = iota
	recognizeTime
	recognizeAny
	recognizeError
	recognizeRawMessage
	// recognizeStdUUID is an identity match on the go1.27 stdlib uuid.UUID.
	//
	// Safe everywhere, so it lives in the canonical set applied by [ApplyStdlibSpecials].
	// See [§special-types](./README.md#special-types).
	recognizeStdUUID
	// recognizeUUID is a fuzzy name-only match (case-insensitive "uuid").
	//
	// Caller-gated — opt in only where the type is guaranteed to render as text.
	// See [§special-types](./README.md#special-types).
	recognizeUUID
	// recognizeOpaqueStream is an identity match on the known byte-stream carriers (io.Reader and
	// friends, multipart.File, runtime.NamedReadCloser).
	//
	// Safe everywhere, so it lives in the canonical set applied by [ApplyStdlibSpecials].
	// See [§opaque-streams](./README.md#opaque-streams).
	recognizeOpaqueStream
)

// ApplyStdlibSpecials runs the canonical safe set of identity-based recognizers (any / time.Time /
// error / json.RawMessage / go1.27 uuid.UUID).
//
// Safe at every call site that handles a *types.TypeName. Exported for the parameters and responses
// builders, which reach it from their own field arms: each used to carry a hand-rolled subset of
// these, differing per function, and the subsets drifted.
//
// Call it BEFORE any declaration lookup. Every recognizer here answers from the object's identity
// alone, so requiring a declaration first subordinates a rule that needs nothing to one that can
// fail — and for the predeclared `error` it always fails, since a predeclared object has no package
// and therefore no declaring source to find.
//
// # Details
//
// See [§special-types](./README.md#special-types).
func ApplyStdlibSpecials(obj *types.TypeName, target ifaces.SwaggerTypable, skipExt bool) bool {
	return applySpecialType(obj, target, skipExt,
		recognizeAny, recognizeTime, recognizeError, recognizeRawMessage, recognizeStdUUID,
		recognizeOpaqueStream)
}

// applySpecialType iterates wanted recognizers in order and applies the first match to target,
// returning resolved=true.
//
// Recognizers are identity-based except recognizeUUID, which is fuzzy (caller-gated). skipExt gates
// vendor-extension writes.
//
// # Details
//
// See [§special-types](./README.md#special-types), [§user-overrides](./README.md#user-overrides)
// (skipExt plumbing) and [§quirks](./README.md#quirks) (per-recognizer rationale).
func applySpecialType(obj *types.TypeName, target ifaces.SwaggerTypable, skipExt bool, wanted ...recognizeType) (resolved bool) {
	for _, typeKey := range wanted {
		if resolved := applySpecialTypeForKey(obj, target, skipExt, typeKey); resolved {
			return true
		}
	}

	return false
}

func applySpecialTypeForKey(obj *types.TypeName, target ifaces.SwaggerTypable, skipExt bool, typeKey recognizeType) (resolved bool) {
	switch typeKey {
	case recognizeTime: // special case of the "time.Time" type
		if resolvers.IsStdTime(obj) {
			target.Typed("string", "date-time")

			return true
		}

	case recognizeAny: // e.g type X any or type X interface{}
		if resolvers.IsAny(obj) {
			_ = target.Schema()

			return true
		}

	case recognizeError: // predeclared error; see [§quirks](./README.md#quirks) for x-go-type rationale.
		if resolvers.IsStdError(obj) {
			if !skipExt {
				target.AddExtension("x-go-type", obj.Name())
			}
			target.Typed("string", "")
			return true
		}

	case recognizeRawMessage: // json.RawMessage; see [§quirks](./README.md#quirks) for the "any" rationale.
		if resolvers.IsStdJSONRawMessage(obj) {
			_ = target.Schema()
			return true
		}

	case recognizeStdUUID: // identity — go1.27 stdlib uuid.UUID.
		if resolvers.IsStdUUID(obj) {
			target.Typed("string", "uuid")
			return true
		}

	case recognizeOpaqueStream: // identity — see [§opaque-streams](./README.md#opaque-streams).
		if resolvers.IsOpaqueStream(obj) {
			// x-go-type records which stream this was, because neither answer below can: `byte` says
			// base64 bytes and `file` says an upload, and every recognized type collapses onto the
			// same schema either way. That is the `recognizeError` criterion — stamp when the
			// rendering erases the type — and not the `time.Time` / uuid one, where the format IS
			// the type. See [§traceability](./README.md#traceability).
			if !skipExt {
				target.AddExtension("x-go-type", obj.Pkg().Path()+"."+obj.Name())
			}
			// The only two answers a stream can honestly take, chosen by what the position permits
			// rather than by anything in the declaration. `file` is legal on a formData parameter and
			// nowhere else in OAS 2.0, and it is the canonical upload shape; everywhere else the
			// stream is base64 in a string, which is how OAS 2.0 spells "opaque bytes".
			if target.In() == inFormData {
				target.Typed("file", "")
				return true
			}
			target.Typed("string", "byte")
			return true
		}

	case recognizeUUID: // fuzzy — see [§special-types](./README.md#special-types).
		if obj != nil && strings.ToLower(obj.Name()) == "uuid" {
			target.Typed("string", "uuid")
			return true
		}

	default:
		// ignored
	}

	return false
}
