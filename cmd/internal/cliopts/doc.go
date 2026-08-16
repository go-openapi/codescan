// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package cliopts declares codescan's option surface once, as flags.
//
// Every command that scans - genspec, genspec-wasi - needs the same thing: a [flag.FlagSet] carrying
// one flag per knob on codescan.Options, and a way to copy what was parsed onto the options a scan
// runs with. Written per command, that mapping is partial in a different way each time, and a knob
// added to the library reaches whichever command somebody remembered.
//
// So it is written here instead, as tables keyed by the field each entry writes to, with one guard
// (see the package tests) asserting that no value-typed field of the options is left unreachable.
// A command registers the whole surface and adds only what is its own - where to write, what format,
// how loud.
//
// # Naming
//
// A flag is the kebab-case of the field it writes, without exception: SkipJSONifyInterfaceMethods is
// -skip-jsonify-interface-methods. Guessing a shorter spelling is how a caller ends up guessing
// wrong, and no rule can derive that JSONify is one word where HTTPServer is two - which is also why
// coverage is decided by writing through a setter and seeing what moved, never by mangling a name.
//
// # What is not here
//
// Options that are not values: the packages to scan (positional arguments), a filesystem or export
// data to read (the command opens a path), a specification to merge into (the command loads it), and
// the diagnostic and provenance callbacks. Those are the command's business; the tests carry the
// list, with a reason each.
package cliopts
