// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import "errors"

// ErrBadFlag is a flag whose value is not one of the ones it accepts.
//
// A sentinel rather than a formatted string at each site: a caller - a test, or a command deciding
// an exit code - can then ask which kind of refusal it met without matching on prose that is free to
// change.
var ErrBadFlag = errors.New("bad flag")
