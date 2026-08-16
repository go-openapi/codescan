// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_description_yaml_fence_absorb witnesses the
// post-M6.5-C behaviour when a route's prose carries a stray `---`
// line. Grammar's lexer treats `---` as a YAML fence opener
// regardless of which annotation is being parsed — so lines AFTER
// the fence are captured as raw YAML body until the matching
// closing `---`, vanishing from Title / Description.
//
// QUIRK: routes don't support YAML blocks (the convention is to use
// markdown HRs sparingly and lean on blank-line paragraph breaks for
// structure). The legacy routes parser silently stripped `---` to
// empty via trimCommentPrefix, making this corner case invisible;
// the new path treats routes consistently with every other annotation
// family. Authors who want a horizontal rule should use another form
// (e.g., a long em-dash run) or accept the loss of subsequent prose
// to the YAML capture.
package routes_description_yaml_fence_absorb

/* GetThing swagger:route GET /things/{id} things getThing

Get a thing.

Some intro prose here.

---

Hidden behind the YAML fence — this prose is absorbed as YAML body
and never reaches the Description.

Responses:
  200: description: OK
*/
func GetThing() {}
