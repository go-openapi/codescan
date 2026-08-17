#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
# SPDX-License-Identifier: Apache-2.0
#
# Runs the benchmark matrix over the corpora shipped with the repository: every configuration of the
# working tree, plus one probe per historic release, warm and cold.
#
# See ../README.md. The short version:
#
#   ./run.sh                                          # warm matrix
#   CODESCAN_BENCH_COLD=1 ./run.sh                    # add the cold pass (slow: it compiles closures)
#   CODESCAN_BENCH_HISTORY="v0.33.3 v0.35.1" ./run.sh # which releases to compare against
#
# Results land in results/ as JSON lines and are tabulated at the end; re-running -summarize over
# them costs nothing, so a run can be re-read without being repeated.

set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../../.." && pwd)"
work="$here/.work"
results="$here/results"

# The releases the working tree is compared against, oldest first. The first one is what deltas are
# computed against, so it is the start of the story rather than an arbitrary reference.
read -r -a history <<<"${CODESCAN_BENCH_HISTORY:-v0.33.3 v0.35.1}"
rounds="${CODESCAN_BENCH_ROUNDS:-3}"
pattern="${CODESCAN_BENCH_PATTERN:-./...}"

mkdir -p "$work" "$results"

# --- corpora -----------------------------------------------------------------------------------
#
# Both trees ship in internal/benchmarks/testdata/corpus.tgz and unpack on demand, so a measurement
# needs a clone and nothing else. They vendor their dependencies, so no download either.
echo "corpora:"
corpora=()
while IFS=$'\t' read -r name dir; do
	[[ -n "$name" ]] || continue
	corpora+=("$name:$dir")
	echo "  $name -> $dir"
done < <(cd "$repo" && go run ./internal/benchmarks/corpus/unpack)

# An extra tree to measure alongside them, for the times the question is about somebody else's code.
if [[ -n "${CODESCAN_BENCH_EXTRA_CORPUS:-}" ]]; then
	corpora+=("${CODESCAN_BENCH_EXTRA_CORPUS}")
	echo "  ${CODESCAN_BENCH_EXTRA_CORPUS%%:*} -> ${CODESCAN_BENCH_EXTRA_CORPUS#*:} (extra)"
fi

if [[ ${#corpora[@]} -eq 0 ]]; then
	echo "no corpus resolved; nothing to measure" >&2
	exit 1
fi

# --- builds ------------------------------------------------------------------------------------
#
# The current build comes from this module. Each historic probe needs its own throwaway module
# requiring that release, and it MUST be built with GOWORK=off: the workspace at the repo root would
# substitute the local tree for the released one and quietly measure the working tree every time.
echo
echo "building current tree..."
(cd "$repo" && go build -o "$work/probe-current" ./internal/benchmarks/loader-benchmark)

# label:binary:flags. A historic probe has exactly one row, because the release it links against has
# exactly one way to load and scan — which is what makes it a point of reference.
configs=()

for version in "${history[@]}"; do
	echo "building $version..."
	mkdir -p "$work/$version"
	cp "$here"/*.go "$work/$version/"
	cat >"$work/$version/go.mod" <<EOF
module codescan-benchmark-$version

go 1.25.0

require github.com/go-openapi/codescan $version
EOF
	(
		cd "$work/$version"
		GOWORK=off go mod tidy >/dev/null 2>&1
		GOWORK=off go build -tags baseline -o "$work/probe-$version" .
	)
	configs+=("$version:$work/probe-$version:")
done

current="$work/probe-current"

# The working tree, in the configurations worth telling apart. `current` is the source-loading scan,
# which is what a plain run does; `current+compiled-deps` opts into export data. Rows are labelled by
# configuration rather than by default, so they stay comparable across a release that moves the
# default -- and one has moved.
configs+=(
	"current:$current:"
	"current+toolchain-free:$current:-toolchain-free"
	"current+compiled-deps:$current:-compiled-deps"
)

baseline="${history[0]:-current}"

measure() { # corpus dir label binary flags cache out
	local corpus="$1" dir="$2" label="$3" bin="$4" flags="$5" cache="$6" out="$7"
	# shellcheck disable=SC2086 # flags is a deliberately word-split option list
	"$bin" -dir "$dir" -pattern "$pattern" -corpus "$corpus" -label "$label" -cache "$cache" $flags >>"$out"
}

warm_out="$results/warm.jsonl"
: >"$warm_out"

for entry in "${corpora[@]}"; do
	corpus="${entry%%:*}"
	dir="${entry#*:}"

	echo
	echo "=== $corpus (warm) ==="
	echo "  warming each configuration (discarded)..."
	for cfg in "${configs[@]}"; do
		label="${cfg%%:*}"
		rest="${cfg#*:}"
		measure "$corpus" "$dir" "$label" "${rest%%:*}" "${rest#*:}" warm /dev/null
	done

	for ((i = 1; i <= rounds; i++)); do
		# Alternating within a round, not config-by-config: cache and thermal drift then land on
		# every configuration equally instead of on whichever ran last.
		for cfg in "${configs[@]}"; do
			label="${cfg%%:*}"
			rest="${cfg#*:}"
			measure "$corpus" "$dir" "$label" "${rest%%:*}" "${rest#*:}" warm "$warm_out"
		done
		echo "  round $i/$rounds"
	done
done

# --- the cold pass -----------------------------------------------------------------------------
#
# Opt-in because it is slow by construction: each cell starts from an empty build cache, and the
# export-data configuration compiles the corpus's whole dependency closure to fill it.
#
# GOMODCACHE is left alone, so this isolates the BUILD cache and nothing else. The private GOCACHE
# means the operator's own cache is never touched.
if [[ "${CODESCAN_BENCH_COLD:-}" == "1" ]]; then
	cold_out="$results/cold.jsonl"
	: >"$cold_out"

	for entry in "${corpora[@]}"; do
		corpus="${entry%%:*}"
		dir="${entry#*:}"

		echo
		echo "=== $corpus (cold) ==="
		for cfg in "${configs[@]}"; do
			label="${cfg%%:*}"
			rest="${cfg#*:}"
			cold="$work/cold-cache"
			rm -rf "$cold"
			mkdir -p "$cold"
			GOCACHE="$cold" measure "$corpus" "$dir" "$label" "${rest%%:*}" "${rest#*:}" cold "$cold_out"
			echo "  $label -> build cache produced: $(du -sh "$cold" 2>/dev/null | cut -f1)"
			rm -rf "$cold"
		done
	done
fi

# --- report ------------------------------------------------------------------------------------
echo
echo "================================ results ================================"
files="$warm_out"
[[ -f "$results/cold.jsonl" ]] && files="$files,$results/cold.jsonl"
"$current" -summarize "$files" -baseline-label "$baseline"
echo
echo "raw measurements: $results/"
