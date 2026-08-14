// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package runstats shows what the last scan cost: the wall clock, and the memory the process moved across the run.
//
// The measurement itself belongs to the scan package (see [scan.Cost], which states what the figures do and do not
// mean); this package only lays them out. The split is deliberate - the fences have to be taken around the work, on
// the goroutine doing it, while what a reader is told about them is a rendering decision.
package runstats
