// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package vfs

import (
	"testing"
	"testing/fstest"

	"github.com/go-openapi/testify/v2/assert"
)

// A path handed to a virtual filesystem is normalised by its own shape, not by the host's.
//
// The two forms have to land in the same place: io/fs holds one rooted namespace, and a caller who writes an absolute
// path means the top of it either way. A Windows drive is the case that does not follow from the POSIX rule — it is
// what makes the path absolute and it is not a separator — so "D:\a\b" left as "D:/a/b" addressed nothing, and every
// read under a scan pointed at such a path failed.
func TestCleanNormalisesEveryRootedForm(t *testing.T) {
	t.Parallel()

	v := New(fstest.MapFS{})

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"posix absolute", "/a/b", "a/b"},
		{"posix relative", "./a/b", "a/b"},
		{"windows drive", `D:\a\b`, "a/b"},
		{"windows drive, slashes", "D:/a/b", "a/b"},
		{"windows drive, lowercase", `c:\a\b`, "a/b"},
		{"windows relative", `a\b`, "a/b"},
		{"drive root", `D:\`, "."},
		{"empty", "", "."},
		{"root", "/", "."},
		{"colon that is not a drive", "ab:c/d", "ab:c/d"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, v.Clean(tc.in))
		})
	}
}

// Naming the last element of a directory is how the walk recognises a vendor directory, so it has to work whichever
// separator the path carries: path.Base looks for a forward slash and hands a Windows path back whole, which reads as
// "this directory is not called vendor" and lets the wildcard walk straight into it.
func TestBaseNamesTheLastElement(t *testing.T) {
	t.Parallel()

	v := New(fstest.MapFS{})

	assert.Equal(t, "vendor", v.Base("/src/m/vendor"))
	assert.Equal(t, "vendor", v.Base(`D:\src\m\vendor`))
	assert.Equal(t, "m", v.Base("src/m"))
}

// A walk hands paths back in the form the caller wrote them, because they become the file names in every position the
// scan reports. The walk itself happens in io/fs's slash-separated namespace, so the caller's separator has to be put
// back on the way out.
func TestRebaseKeepsTheCallersForm(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		root, inner, out string
		want             string
	}{
		{"posix absolute", "/src/m", "src/m", "src/m/sub", "/src/m/sub"},
		{"posix root", "/", ".", "sub/pkg", "/sub/pkg"},
		{"relative", ".", ".", "sub", "./sub"},
		{"the root itself", "/src/m", "src/m", "src/m", "/src/m"},
		{"windows drive", `D:\src\m`, "src/m", "src/m/sub/pkg", `D:\src\m\sub\pkg`},
		{"windows drive root", `D:\`, ".", "sub", `D:\sub`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, rebase(tc.root, tc.inner, tc.out))
		})
	}
}
