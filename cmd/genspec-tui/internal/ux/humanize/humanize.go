// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package humanize

import (
	"fmt"
	"time"
)

// Duration renders d compactly: "947ms", "3s", "1m 3s" (minute form drops a zero-second remainder, e.g. "2m").
func Duration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
	default:
		d = d.Round(time.Second)
		mins := int(d / time.Minute)
		secs := int((d % time.Minute) / time.Second)
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}

		return fmt.Sprintf("%dm %ds", mins, secs)
	}
}

// The binary units memory is reported in. A megabyte here is 1024 KB, matching what every other tool a reader would
// compare this against (top, ps, the Go runtime's own docs) means by MB.
const (
	kb = 1 << 10
	mb = 1 << 20
	gb = 1 << 30
)

// Bytes renders a size: "912 B", "44 KB", "412.3 MB", "1.21 GB".
//
// Precision grows with the unit rather than staying fixed, because the reader's question changes with it: at kilobytes
// nobody cares about a fraction, while at gigabytes two decimals are the difference between two readings.
func Bytes(n uint64) string {
	switch {
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < mb:
		return fmt.Sprintf("%.0f KB", float64(n)/kb)
	case n < gb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/gb)
	}
}

// SignedBytes renders a difference, always carrying its sign: "+38.0 MB", "-4.1 MB", "+0 B".
//
// The sign is explicit even when positive: these are read as changes, and a bare number invites being read as a total.
func SignedBytes(n int64) string {
	if n >= 0 {
		return "+" + Bytes(uint64(n))
	}

	// Negating MinInt64 overflows. Adding one before the negation keeps it in range, at the cost of a single byte at a
	// magnitude no heap difference can reach.
	//
	//nolint:gosec // -(n+1) is non-negative for every n < 0, which is the only branch this is reached on
	return "-" + Bytes(uint64(-(n+1))+1)
}

// SignedCount renders a count difference in short form: "+410 k", "-1.2 M", "+37".
func SignedCount(n int64) string {
	sign := "+"
	if n < 0 {
		sign = "-"
		n = -n
	}

	switch {
	case n < 1_000:
		return fmt.Sprintf("%s%d", sign, n)
	case n < 1_000_000:
		return fmt.Sprintf("%s%.0f k", sign, float64(n)/1_000)
	default:
		return fmt.Sprintf("%s%.1f M", sign, float64(n)/1_000_000)
	}
}
