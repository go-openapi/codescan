#!/usr/bin/env bash
# SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
# SPDX-License-Identifier: Apache-2.0
#
# Runs the loader benchmark matrix: a released baseline against the working tree, over every corpus
# that is present, warm and cold.
#
# See README.md. The short version:
#
#   CODESCAN_BENCH_DOCKERCTL_DIR=... ./run.sh              # warm matrix over whatever is configured
#   CODESCAN_BENCH_COLD=1 ./run.sh                         # add the cold pass (slow: it compiles closures)
#   CODESCAN_BENCH_BASELINE=v0.36.3 ./run.sh               # compare against a different release
#
# Results land in results/ as JSON lines and are tabulated at the end; re-running -summarize over
# them costs nothing, so a run can be re-read without being repeated.

set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
work="$here/.work"
results="$here/results"

baseline="${CODESCAN_BENCH_BASELINE:-v0.36.3}"
rounds="${CODESCAN_BENCH_ROUNDS:-3}"
pattern="${CODESCAN_BENCH_PATTERN:-./...}"

mkdir -p "$work" "$results"

# --- corpora -----------------------------------------------------------------------------------
#
# Every corpus is external and large; none is vendored into this repo, because the point of measuring
# them is that they are real trees. Name them through the environment — there is deliberately no
# default path, so a missing corpus is skipped by name rather than silently measuring someone else's
# checkout.
corpora=()
add_corpus() { # name env-var
	local dir="${!2:-}"
	if [[ -n "$dir" && -d "$dir" ]]; then
		corpora+=("$1:$dir")
	else
		echo "  skipping corpus '$1' (set $2 to a checkout to include it)"
	fi
}

echo "corpora:"
add_corpus dockerctl CODESCAN_BENCH_DOCKERCTL_DIR
add_corpus kubeapi CODESCAN_BENCH_KUBEAPI_DIR
add_corpus kubeapi-vendored CODESCAN_BENCH_KUBEAPI_VENDORED_DIR

if [[ ${#corpora[@]} -eq 0 ]]; then
	echo "no corpus configured; nothing to measure" >&2
	exit 1
fi

# --- builds ------------------------------------------------------------------------------------
#
# The current build comes from this module. The baseline needs its own module requiring the released
# library, and it MUST be built with GOWORK=off: the workspace at the repo root would substitute the
# local tree for the released one and quietly measure the working tree twice.
echo
echo "building current tree..."
(cd "$repo" && go build -o "$work/loader-benchmark-current" ./hack/loader-benchmark)

echo "building baseline $baseline..."
mkdir -p "$work/baseline"
cp "$here"/*.go "$work/baseline/"
cat >"$work/baseline/go.mod" <<EOF
module codescan-loader-benchmark-baseline

go 1.25.0

require github.com/go-openapi/codescan $baseline
EOF
(
	cd "$work/baseline"
	GOWORK=off go mod tidy >/dev/null 2>&1
	GOWORK=off go build -tags baseline -o "$work/loader-benchmark-baseline" .
)

current="$work/loader-benchmark-current"
base="$work/loader-benchmark-baseline"

# --- the matrix --------------------------------------------------------------------------------
#
# label:binary:flags. The baseline has exactly one row because the release has exactly one way to
# load and scan — which is what makes it a baseline.
configs=(
	"$baseline:$base:"
	"current:$current:"
	"current+toolchain-free:$current:-toolchain-free"
	"current+compiled-deps:$current:-compiled-deps"
)

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
