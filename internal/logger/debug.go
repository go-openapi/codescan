// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"log"
)

// logCallerDepth is the caller depth for log.Output.
const logCallerDepth = 2

func DebugLogf(debug bool, format string, args ...any) {
	if debug {
		_ = log.Output(logCallerDepth, fmt.Sprintf(format, args...))
	}
}
