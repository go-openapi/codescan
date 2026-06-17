// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// TestConcatScore locks the production readability score (reduce.go) that the
// name-identity concat budget is checked against. The exploration harness in
// concat_score_test.go (disabled) re-challenges it; this test pins the
// contract K3's hierarchical fallback will rely on.
func TestConcatScore(t *testing.T) {
	t.Run("empty is perfectly readable", func(t *testing.T) {
		assert.Equal(t, 0.0, concatScore("", nil))
	})

	t.Run("more than three parts is always ruled out", func(t *testing.T) {
		assert.Equal(t, 1.0, concatScore("CatHouseGardenElephant",
			[]string{"cat", "house", "garden", "elephant"}))
	})

	t.Run("short two-part concat is comfortably readable", func(t *testing.T) {
		// 8/24*0.5 + 0.5*0.3 + 5/12*0.2 = 0.4 exactly.
		assert.InDelta(t, 0.40, concatScore("CatHouse", []string{"cat", "house"}), 1e-9)
	})

	t.Run("a longer/more-parts concat scores worse", func(t *testing.T) {
		short := concatScore("CatHouse", []string{"cat", "house"})
		longer := concatScore("HouseGardenElephant", []string{"house", "garden", "elephant"})
		assert.Less(t, short, longer, "more parts and length => higher (worse) score")
	})

	t.Run("score is clamped to 1.0", func(t *testing.T) {
		assert.Equal(t, 1.0, concatScore("GardenElephantConstruction",
			[]string{"garden", "elephant", "construction"}))
	})
}

// TestResolveHierarchicalGroup locks the nested-path resolution logic the
// integration fixture doesn't reach: depth-deepening on a shared parent, and
// the taken-vs-shareable root distinction.
func TestResolveHierarchicalGroup(t *testing.T) {
	none := func() map[string]struct{} { return map[string]struct{}{} }

	t.Run("distinct package leaves resolve at depth 1", func(t *testing.T) {
		paths := resolveHierarchicalGroup([]string{"x/a/Config", "x/b/Config"}, none(), none())
		assert.Equal(t, []string{"a", "Config"}, paths["x/a/Config"])
		assert.Equal(t, []string{"b", "Config"}, paths["x/b/Config"])
	})

	t.Run("shared immediate parent deepens to a distinct grandparent", func(t *testing.T) {
		paths := resolveHierarchicalGroup([]string{"x/a/mongo/Book", "x/b/mongo/Book"}, none(), none())
		assert.Equal(t, []string{"a", "mongo", "Book"}, paths["x/a/mongo/Book"])
		assert.Equal(t, []string{"b", "mongo", "Book"}, paths["x/b/mongo/Book"])
	})

	t.Run("a root clashing with a flat name forces deepening", func(t *testing.T) {
		taken := map[string]struct{}{"shared": {}} // a flat definition is named "shared"
		paths := resolveHierarchicalGroup([]string{"x/shared/Config", "y/other/Config"}, taken, none())
		assert.Equal(t, []string{"x", "shared", "Config"}, paths["x/shared/Config"])
		assert.Equal(t, []string{"y", "other", "Config"}, paths["y/other/Config"])
	})

	t.Run("a shareable container root is reused without deepening", func(t *testing.T) {
		taken := map[string]struct{}{"shared": {}}
		shareable := map[string]struct{}{"shared": {}} // "shared" is itself a container
		paths := resolveHierarchicalGroup([]string{"x/shared/Config", "y/other/Config"}, taken, shareable)
		assert.Equal(t, []string{"shared", "Config"}, paths["x/shared/Config"])
		assert.Equal(t, []string{"other", "Config"}, paths["y/other/Config"])
	})
}

// TestConcatScoreVersusDefaultBudget shows the default budget cleanly separates
// the common short concats (accepted) from the long ones (Stage-3 candidates).
func TestConcatScoreVersusDefaultBudget(t *testing.T) {
	accepted := concatScore("BWidget", []string{"b", "widget"})
	assert.Less(t, accepted, defaultNameConcatBudget,
		"a typical 2-segment concat stays under the default budget")

	candidate := concatScore("GardenElephantConstruction",
		[]string{"garden", "elephant", "construction"})
	assert.Greater(t, candidate, defaultNameConcatBudget,
		"a long 3-segment concat exceeds the default budget (Stage-3 candidate)")
}
