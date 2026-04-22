// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package classify

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestIsAllowedExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ext  string
		want bool
	}{
		{"x-foo", true},
		{"X-bar", true},
		{"x-", true},
		{"y-foo", false},
		{"foo", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.ext, func(t *testing.T) {
			assert.EqualT(t, tc.want, IsAllowedExtension(tc.ext))
		})
	}
}
