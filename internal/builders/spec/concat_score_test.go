// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

// concat_score_test.go is the K2 exploration harness for the readability score that gates
// name-identity concat names (see reduce.go concatScore and Options.NameConcatBudget).
//
// It is NOT a regression test and is DISABLED by default (t.SkipNow): it logs a ranked table for
// human judgement and exists so we can re-challenge the production score function against
// alternatives in the future.
//
// To re-run it, comment out the SkipNow and:
//
// 	go test ./internal/builders/spec/ -run TestConcatScoreExperiment -v
//
// The corpus, the combinations, and the row order are all deterministic, so the output is stable
// across runs.

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/go-openapi/swag/mangling"
)

// scoreFunc is a named candidate readability score.
//
// Add more here to compare them side by side in the table — every function gets its own column.
// The "weighted" candidate is the production concatScore (reduce.go), so this harness always
// re-challenges whatever ships.
type scoreFunc struct {
	name string
	fn   func(concat string, idents []string) float64
}

// scoreSeed is Fred's seed function, transcribed faithfully with three fixes
// needed to compile/run:
//   - `l := float64(len(ident)) > maxIdentLen` assigned a bool; split into an
//     assignment followed by the comparison.
//   - the final `max(1.00, …)` is changed to `min(1.00, …)`: the score is
//     documented as [0,1] with "lower = more readable", so 1.0 must be a
//     CEILING, not a floor (max would pin everything to ≥1.0).
//   - added the missing closing paren / func header.
//
// It is kept only as the cautionary baseline: its /maxWord^2 term rewards a single dominating long
// word, so it anti-correlates with intuition.
func scoreSeed(concat string, idents []string) float64 {
	const maxParts = 3.00

	if len(idents) == 0 {
		return 0.00
	}

	if len(idents) > int(maxParts) {
		return 1.00
	}

	lenConcat := float64(len(concat))
	seqLen := float64(len(idents))
	maxIdent := -1.00

	for _, ident := range idents {
		if l := float64(len(ident)); l > maxIdent {
			maxIdent = l
		}
	}

	return min(1.00, (lenConcat*seqLen/maxIdent)/(maxParts*maxIdent))
}

// scoreLinear is a contrast baseline: monotonic in total concat length only.
func scoreLinear(concat string, idents []string) float64 {
	const (
		maxParts   = 3.00
		wordBudget = 12.00
	)

	if len(idents) == 0 {
		return 0.00
	}

	if len(idents) > int(maxParts) {
		return 1.00
	}

	return min(1.00, float64(len(concat))/(maxParts*wordBudget))
}

func maxIdentLen(idents []string) int {
	m := 0
	for _, ident := range idents {
		if l := len(ident); l > m {
			m = l
		}
	}

	return m
}

// combinations returns all k-subsets of [0,n) in lexicographic index order (deterministic).
func combinations(n, k int) [][]int {
	if k <= 0 || k > n {
		return nil
	}

	idx := make([]int, k)
	for i := range idx {
		idx[i] = i
	}

	var res [][]int
	for {
		comb := make([]int, k)
		copy(comb, idx)
		res = append(res, comb)

		i := k - 1
		for i >= 0 && idx[i] == n-k+i {
			i--
		}
		if i < 0 {
			break
		}

		idx[i]++
		for j := i + 1; j < k; j++ {
			idx[j] = idx[j-1] + 1
		}
	}

	return res
}

func TestConcatScoreExperiment(t *testing.T) {
	t.SkipNow() // exploration harness, not a regression test — see file header.

	// concatScoreCorpus: five common English words spanning 3..12 characters, so a 2/3-word concat
	// exercises a wide spread of part lengths.
	concatScoreCorpus := []string{
		"cat",          // 3
		"house",        // 5
		"garden",       // 6
		"elephant",     // 8
		"construction", // 12
	}

	m := mangling.NewNameMangler()

	funcs := []scoreFunc{
		{"seed", scoreSeed},
		{"linear", scoreLinear},
		{"weighted", concatScore}, // the production function (reduce.go)
	}

	type row struct {
		idents []string
		concat string
	}

	var rows []row
	add := func(idxs []int) {
		idents := make([]string, len(idxs))
		for i, ix := range idxs {
			idents[i] = concatScoreCorpus[ix]
		}
		concat := m.ToGoName(strings.Join(idents, " "))
		rows = append(rows, row{idents: idents, concat: concat})
	}

	for _, c := range combinations(len(concatScoreCorpus), 2) {
		add(c)
	}
	for _, c := range combinations(len(concatScoreCorpus), 3) {
		add(c)
	}
	add([]int{0, 1, 2, 3}) // one 4-word sample: ruled out by maxParts (score 1.0)

	if len(rows) == 0 {
		t.Fatal("no rows generated")
	}

	// Rank by the weighted (production) score, ascending = most readable first; tie-break by concat
	// for stability.
	sort.SliceStable(rows, func(i, j int) bool {
		si, sj := concatScore(rows[i].concat, rows[i].idents), concatScore(rows[j].concat, rows[j].idents)
		if si != sj {
			return si < sj
		}
		return rows[i].concat < rows[j].concat
	})

	var b strings.Builder
	fmt.Fprintf(&b, "\nconcat readability scores (lower = more readable), ranked by weighted\n\n")
	fmt.Fprintf(&b, "%-30s %-24s %4s %5s %3s", "idents", "ToGoName(concat)", "len", "maxW", "n")
	for _, f := range funcs {
		fmt.Fprintf(&b, " %8s", f.name)
	}
	b.WriteByte('\n')

	for _, r := range rows {
		fmt.Fprintf(&b, "%-30s %-24s %4d %5d %3d",
			strings.Join(r.idents, "+"), r.concat, len(r.concat), maxIdentLen(r.idents), len(r.idents))
		for _, f := range funcs {
			fmt.Fprintf(&b, " %8.3f", f.fn(r.concat, r.idents))
		}
		b.WriteByte('\n')
	}

	t.Log(b.String())
}
