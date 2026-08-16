// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// The output half of the on-demand scanner's control loop.
//
// Cost alone is not a control loop. A load that is cheaper AND quietly drops a `swagger:strfmt`
// format is exactly what this stream must not ship, so every configuration that changes HOW
// dependency types arrive is A/B'd against the ordinary one over the fixture corpus.
//
// The pattern comes from TestLoaderChoice_AgreeOnTheRealFilesystem: compare WHOLE marshalled
// documents rather than spot-checking, because anything less would pass while a property quietly
// went missing. What is added here is breadth on both sides of the matrix — the whole fixture
// corpus rather than one fixture, and the dependency-type configurations rather than one loader
// flag — and an explicit expectation for the cases where a configuration is KNOWN to differ.
//
// A known difference is asserted as a difference, not tolerated. When the stream closes one of
// these gaps the assertion fails, which is the point: an expectation that silently survives its own
// fix is not an expectation.
//
// Two tiers, because a full sweep costs minutes:
//
//	default            the curated targets below — the fixtures whose meaning comes from a
//	                   dependency, which is where these configurations can bite
//	CODESCAN_AB_CORPUS=1   every fixture bundle under testdata/{enhancements,bugs,goparsing}
//
// Set CODESCAN_AB_REPORT=1 to print the divergences instead of asserting them, which is how the
// expectation table below is regenerated.

import (
	"archive/zip"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const (
	// envABCorpus widens the A/B from the curated targets to every fixture bundle.
	envABCorpus = "CODESCAN_AB_CORPUS"

	// envABReport prints divergences instead of asserting them, to regenerate abExpected.
	envABReport = "CODESCAN_AB_REPORT"

	// envABExportData points at an export-data blob covering the fixtures module, which is what the
	// export-data configuration needs. Without it that configuration is not compared.
	//
	//	go run ./hack/genexportdata -dir testdata -out /tmp/testdata-exportdata.zip std
	//
	// `std` rather than `./...`: the wider pattern needs `go list -export` to BUILD the fixtures'
	// dependency closure, and testdata/go.sum is missing go.mod entries for two of go-openapi/swag's
	// submodules, so it fails before writing anything. std is also the interesting half — it is the
	// bulk of any closure and carries no annotations, so it is what export data is for.
	envABExportData = "CODESCAN_AB_EXPORTDATA"
)

// abConfig is one way of supplying dependency types, named so a divergence can be attributed.
type abConfig struct {
	name  string
	apply func(*codescan.Options)
}

// abBaseline is the reference every configuration is compared against: every dependency read from
// source, nothing taken from a compiler's word for it.
//
// Deliberately not the default Options. Since v0.36.4 the default takes dependency types from export
// data, and a test asking "does this shortcut change the document?" has to hold the no-shortcut scan
// as its reference — otherwise the shortcut is on both sides of the comparison and the question
// answers itself.
func abBaseline(o *codescan.Options) { o.SkipCompiledDependencies = true }

// abConfigs returns the configurations to compare against [abBaseline].
func abConfigs(tb testing.TB) []abConfig {
	tb.Helper()

	configs := []abConfig{
		{
			// Axis A. Already covered for the petstore by TestLoaderChoice_AgreeOnTheRealFilesystem.
			// It changes the loader, and under it dependency types come from source as they do in the
			// baseline, so it is the control and should never diverge.
			name:  "toolchain-free",
			apply: func(o *codescan.Options) { o.ToolchainFreeLoader = true },
		},
		{
			// Axis B, and since v0.36.4 the default. go/packages takes dependency types from `go list
			// -export` wholesale — a LoadMode is one value for the whole load — and the annotated ones
			// are read back afterwards, which is how it reaches the per-dependency outcome the
			// toolchain-free route decides during it.
			name:  "compiled-dependencies",
			apply: func(*codescan.Options) {},
		},
	}

	if blob := abExportDataBlob(tb); blob != nil {
		configs = append(configs, abConfig{
			// Axis C. The toolchain-free route decides per dependency — one whose source carries
			// annotations is read from source, one that says nothing is taken from export data — so
			// this one is expected to agree everywhere.
			name: "export-data",
			apply: func(o *codescan.Options) {
				o.ToolchainFreeLoader = true
				o.ExportData = blob
			},
		})
	}

	return configs
}

// abExportDataBlob opens the fixtures module's export-data blob, or returns nil saying how to make
// one. It is an input rather than a checked-in artifact: the export format is tied to the Go
// release that produced it.
func abExportDataBlob(tb testing.TB) fs.FS {
	tb.Helper()

	path := os.Getenv(envABExportData)
	if path == "" {
		tb.Logf("export-data configuration not compared: set %s to a blob covering the fixtures module\n"+
			"    go run ./hack/genexportdata -dir testdata -out /tmp/testdata-exportdata.zip std", envABExportData)

		return nil
	}

	// Operator-supplied on purpose — the blob is a build artifact kept outside the repo.
	info, err := os.Stat(path) //nolint:gosec // an operator-supplied measurement input, not user data
	require.NoErrorf(tb, err, "%s=%q", envABExportData, path)

	if info.IsDir() {
		return os.DirFS(path)
	}

	r, err := zip.OpenReader(path)
	require.NoErrorf(tb, err, "%s=%q is neither a directory nor a readable zip", envABExportData, path)
	tb.Cleanup(func() { _ = r.Close() })

	return r
}

// abCuratedTargets are the fixture bundles whose meaning comes from a DEPENDENCY rather than from
// their own source, which is the only place a dependency-type configuration can change the answer.
//
// strfmt is the case the stream turns on: `// swagger:strfmt date-time` sits above strfmt.DateTime
// in strfmt's own source, and that comment is the only reason a field of that type acquires a
// format. The stdlib entries are the other half — types recognised by identity (time.Time,
// json.RawMessage) or by method set (encoding.TextMarshaler), which export data serves and a stub
// does not.
//
// Every target that abExpected has an entry for is listed here, so no expectation can go stale
// without the default tier noticing. The rest are controls: targets that read a dependency's types
// and must NOT change.
func abCuratedTargets() []string {
	return []string{
		// Every known divergence lives in one of these.
		"./goparsing/petstore/...",
		"./goparsing/classification/...",
		"./goparsing/go123/...",
		"./goparsing/bookings/...",
		"./goparsing/spec/...",
		"./bugs/301/...",
		"./bugs/2248/...",
		"./bugs/3412/...",
		"./enhancements/opaque-streams/...",

		// Controls.
		"./bugs/2871/...",
		"./enhancements/strfmt-arrays/...",
		"./enhancements/strfmt-symmetry-core/...",
		"./enhancements/text-marshal/...",
		"./enhancements/raw-message-override/...",
		"./enhancements/discriminated-subtypes/...",
		"./enhancements/interface-methods/...",
		"./enhancements/alias-calibration-stdlib/...",
	}
}

// abExpected records the (configuration, target) pairs that are KNOWN to produce a different spec, with why.
// Everything not listed must agree exactly. Keys are "<config>|<target>"; regenerate the table with
// CODESCAN_AB_REPORT=1.
//
// It is empty, and staying empty is the contract: how dependency types arrive is a question about cost, not about
// meaning, so a configuration that changes the document is a bug in that configuration. Add a row only with the
// reasoning for why the difference is right, and expect to be asked.
//
// It was six rows, in two families, and both closed the same way — by separating what a dependency SAYS from what
// it DECLARES.
//
//   - A DEPENDENCY'S OWN ANNOTATIONS. `swagger:strfmt` marks live in strfmt's source and export data carries types,
//     not comments. Closed by reading back, after the load, the dependencies whose files carry the marker.
//
//   - A DECLARATION THE SCANNED CODE ASKS FOR. The marker scan is the wrong question here: a model declared in an
//     unannotated dependency, or a stdlib type consumed where no recognizer answers for it, needs source that
//     nothing in that source asked to have read. Closed by fetching it at the lookup — see
//     ScanCtx.readBackOnDemand, and TestDependencyDeclarations_ReadBackOnDemand for the witness.
//
// The obvious version of that second fix is the one that fails, and it is recorded here because it looks right.
// Reading back every non-stdlib dependency up front brings the definition back EMPTY and silences the warning that
// named it, because fields are located by POSITION and a *types.Var out of export data indexes a different
// token.File than the syntax we parsed. It also gives away a third of the wall clock and a third of the peak RSS,
// which is most of the reason to choose the option. Both halves are needed: on demand, and bridged by name and line
// (resolvers.FindASTFieldFor).
//
// The assertion below flips to a failure when a listed row stops diverging, which is how a stale expectation
// announces itself.
func abExpected() map[string]string {
	return map[string]string{}
}

// abKey is the key abExpected is written in.
func abKey(config, target string) string { return config + "|" + target }

// abScan runs one target under one configuration and returns the marshalled document, or the error
// text when the scan fails.
//
// A failure is an outcome to compare like any other: some fixtures are deliberately invalid, and a
// configuration that turns a clean error into a panic — or into a silent success — is exactly the
// kind of regression this is here to catch.
func abScan(tb testing.TB, target string, apply func(*codescan.Options)) string {
	tb.Helper()

	opts := &codescan.Options{
		Packages:   []string{target},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	}
	if apply != nil {
		apply(opts)
	}

	doc, err := codescan.Run(opts)
	if err != nil {
		return "error: " + err.Error()
	}

	blob, err := json.Marshal(doc)
	require.NoError(tb, err)

	return string(blob)
}

// abTargets returns the bundles to compare: the curated list, or every fixture bundle under
// CODESCAN_AB_CORPUS=1.
func abTargets(tb testing.TB) []string {
	tb.Helper()

	if os.Getenv(envABCorpus) != "1" {
		return abCuratedTargets()
	}

	root := scantest.FixturesDir()

	var targets []string
	for _, family := range []string{"enhancements", "bugs", "goparsing"} {
		entries, err := os.ReadDir(filepath.Join(root, family))
		require.NoErrorf(tb, err, "read fixture family %s", family)

		for _, e := range entries {
			if e.IsDir() {
				targets = append(targets, "./"+family+"/"+e.Name()+"/...")
			}
		}
	}
	sort.Strings(targets)

	return targets
}

// TestLoaderConfigs_AgreeOnTheFixtureCorpus is the A/B itself.
//
// Whole documents, every target, every configuration that changes where dependency types come from.
func TestLoaderConfigs_AgreeOnTheFixtureCorpus(t *testing.T) {
	t.Parallel()

	targets := abTargets(t)
	configs := abConfigs(t)
	expected := abExpected()
	report := os.Getenv(envABReport) == "1"

	var mx sync.Mutex
	var diverged []string

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			baseline := abScan(t, target, abBaseline)

			for _, config := range configs {
				got := abScan(t, target, config.apply)
				key := abKey(config.name, target)
				why, expectedToDiffer := expected[key]

				switch {
				case report:
					if !sameSpec(baseline, got) {
						mx.Lock()
						diverged = append(diverged, key)
						mx.Unlock()
					}
				case expectedToDiffer:
					assert.Falsef(t, sameSpec(baseline, got),
						"%s no longer differs from the baseline on %s. The expectation said: %s. "+
							"If that gap has been closed, delete the abExpected entry", config.name, target, why)
				default:
					assert.Truef(t, sameSpec(baseline, got),
						"%s produced a different spec for %s, and no expectation covers it. "+
							"Either it is a regression, or it is a legitimate difference that has to be "+
							"written down in abExpected with a reason", config.name, target)
				}
			}
		})
	}

	if !report {
		return
	}

	// Cleanup so the summary runs after every parallel subtest has finished.
	t.Cleanup(func() {
		sort.Strings(diverged)
		t.Logf("=== divergences (%d) — paste into abExpected ===", len(diverged))
		for _, key := range diverged {
			t.Logf("\t%q: %q,", key, "TODO: why")
		}
	})
}

// sameSpec compares two marshalled documents, tolerating key order.
func sameSpec(a, b string) bool {
	if a == b {
		return true
	}
	if strings.HasPrefix(a, "error: ") || strings.HasPrefix(b, "error: ") {
		return false
	}

	var da, db any
	if json.Unmarshal([]byte(a), &da) != nil || json.Unmarshal([]byte(b), &db) != nil {
		return false
	}

	ja, erra := json.Marshal(da)
	jb, errb := json.Marshal(db)

	return erra == nil && errb == nil && string(ja) == string(jb)
}
