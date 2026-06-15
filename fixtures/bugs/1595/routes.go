// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug1595 reproduces go-swagger#1595: a swagger:operation written in a
// /* */ block comment must scan like its // equivalent.
//
// Both annotations below are gofmt-clean (run gofmt and they don't change), so
// they exercise the forms real source actually takes:
//
//   - displayProduct uses the flush-left block style. gofmt rewrites the
//     column-0 YAML body into its "code block" canonical form — a blank line
//     under `responses:` and TAB-indented children. That form must still parse
//     (the framing strip here composes with yaml.RemoveIndent's dedent/retab,
//     94ec08f). Do NOT "tidy" the tabs back to spaces: gofmt would just rewrite
//     them again.
//   - displayWidget uses the *-decorated block style, which gofmt leaves as-is;
//     it exercises the godoc `* ` continuation strip.
package bug1595

/*
swagger:operation GET /products display displayProduct

Displays a product.

Some description.

---
produces:
- application/json
responses:

	'200':
	  description: Success
*/
func displayProduct() {}

/*
 * swagger:operation GET /widgets display displayWidget
 *
 * Displays a widget.
 *
 * ---
 * produces:
 * - application/json
 * responses:
 *   '200':
 *     description: Success
 */
func displayWidget() {}
