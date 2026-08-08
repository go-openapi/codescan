// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build !exportdata

package exportdata

import "io/fs"

// Embedded reports that this build carries no export data.
func Embedded() (fs.FS, bool) { return nil, false }
