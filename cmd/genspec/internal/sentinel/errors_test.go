// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package sentinel_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// declarable is the point of the string type, checked where it can be: in a const block. A package variable would
// leave the exit status of the command writable by anything that imports this.
const declarable sentinel.Error = sentinel.ErrUsage

// errCause stands for whatever a refusal was actually about, which is what a sentinel is wrapped around.
var errCause = errors.New("-input names a directory")

// These are what the exit status is decided from, so what matters about them is that they survive being wrapped and
// stay distinguishable from one another - which is what errors.Is is asked, all the way up in main.
func TestError(t *testing.T) {
	t.Parallel()

	t.Run("should answer to itself through a wrapping", func(t *testing.T) {
		t.Parallel()

		wrapped := fmt.Errorf("%w: %w", sentinel.ErrUsage, errCause)

		require.ErrorIs(t, wrapped, sentinel.ErrUsage)
		assert.Contains(t, wrapped.Error(), "incorrect usage")
	})

	t.Run("should stay apart from the others", func(t *testing.T) {
		// A command that could not tell these apart would report one exit status for every kind of refusal.
		t.Parallel()

		require.NotErrorIs(t, sentinel.ErrUsage, sentinel.ErrDiagnostics)
		require.NotErrorIs(t, sentinel.ErrDiagnostics, sentinel.ErrInvalidSpec)
		require.NotErrorIs(t, sentinel.ErrInvalidSpec, sentinel.ErrUsage)
	})

	t.Run("should read as what it says", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "incorrect usage", declarable.Error())
		assert.Equal(t, "the specification is not valid", sentinel.ErrInvalidSpec.Error())
	})

	t.Run("should compare on identity, not on wording", func(t *testing.T) {
		// Two errors that read the same are the same, which is what makes a constant usable as a sentinel.
		t.Parallel()

		require.ErrorIs(t, sentinel.Error("incorrect usage"), sentinel.ErrUsage)
		require.NotErrorIs(t, sentinel.Error("Incorrect usage"), sentinel.ErrUsage)
	})
}
