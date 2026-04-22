// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestCollectScannerTitleDescription(t *testing.T) {
	t.Parallel()

	t.Run("title and description separated by blank", func(t *testing.T) {
		headers := []string{
			"// This is the title.",
			"//",
			"// This is the description.",
			"// More description.",
		}
		title, desc := CollectScannerTitleDescription(headers)
		assert.Equal(t, []string{"This is the title."}, title)
		assert.Equal(t, []string{"This is the description.", "More description."}, desc)
	})

	t.Run("title only with punctuation", func(t *testing.T) {
		headers := []string{
			"// A single title line.",
			"// And some description.",
		}
		title, desc := CollectScannerTitleDescription(headers)
		assert.Equal(t, []string{"A single title line."}, title)
		assert.Equal(t, []string{"And some description."}, desc)
	})

	t.Run("title with markdown header prefix", func(t *testing.T) {
		headers := []string{
			"// # My Title",
			"// Description here.",
		}
		title, desc := CollectScannerTitleDescription(headers)
		assert.Equal(t, []string{"My Title"}, title)
		assert.Equal(t, []string{"Description here."}, desc)
	})

	t.Run("no title, all description", func(t *testing.T) {
		headers := []string{
			"// no punctuation at end means no title",
			"// more text",
		}
		title, desc := CollectScannerTitleDescription(headers)
		assert.Empty(t, title)
		assert.Equal(t, []string{"no punctuation at end means no title", "more text"}, desc)
	})

	t.Run("empty", func(t *testing.T) {
		title, desc := CollectScannerTitleDescription(nil)
		assert.Empty(t, title)
		assert.Empty(t, desc)
	})

	t.Run("blank line only", func(t *testing.T) {
		headers := []string{"//"}
		title, desc := CollectScannerTitleDescription(headers)
		assert.Empty(t, title)
		assert.Nil(t, desc)
	})
}
