// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package humanize_test

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/humanize"
)

func TestDuration(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{947 * time.Millisecond, "947ms"},
		{0, "0ms"},
		{time.Second, "1s"},
		{3200 * time.Millisecond, "3s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m"},
		{2 * time.Minute, "2m"},
		{63 * time.Second, "1m 3s"},
	} {
		assert.Equal(t, tc.want, humanize.Duration(tc.in), "%s", tc.in)
	}
}

func TestBytes(t *testing.T) {
	for _, tc := range []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{912, "912 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1024*1024 - 1, "1024 KB"}, // rounds up to the next unit's worth without changing unit
		{1024 * 1024, "1.0 MB"},
		{432 * 1024 * 1024, "432.0 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{3*1024*1024*1024 + 512*1024*1024, "3.50 GB"},
	} {
		assert.Equal(t, tc.want, humanize.Bytes(tc.in), "%d", tc.in)
	}
}

func TestSignedBytes(t *testing.T) {
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "+0 B"},
		{38 * 1024 * 1024, "+38.0 MB"},
		{-4 * 1024 * 1024, "-4.0 MB"},
		{-512, "-512 B"},
	} {
		assert.Equal(t, tc.want, humanize.SignedBytes(tc.in), "%d", tc.in)
	}

	// A heap difference can never reach here, but arithmetic that wraps at the extreme is a bug wherever it lives.
	assert.NotPanics(t, func() { humanize.SignedBytes(-1 << 62) })
}

func TestSignedCount(t *testing.T) {
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "+0"},
		{37, "+37"},
		{-999, "-999"},
		{410_000, "+410 k"},
		{-12_400, "-12 k"},
		{1_250_000, "+1.2 M"},
	} {
		assert.Equal(t, tc.want, humanize.SignedCount(tc.in), "%d", tc.in)
	}
}
