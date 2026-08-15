// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import "runtime/debug"

// Version reports what this build is, as the module system recorded it.
//
// A binary installed with `go install ...@latest` carries its version; one built from a working copy
// carries the revision instead, and says so rather than claiming a release it is not.
func Version(cmd string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return cmd + " (unknown version)"
	}

	if v := info.Main.Version; v != "" && v != "(devel)" {
		return cmd + " " + v
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return cmd + " (devel, " + setting.Value + ")"
		}
	}

	return cmd + " (devel)"
}
