// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestEveryFlagIsAddressableInAConfigFile is the config-file half of the coverage guard, for the
// flags this command adds to the shared ones.
//
// A flag with no section can be typed but not configured, and nothing would notice: from the outside
// that reads exactly like the option not existing.
func TestEveryFlagIsAddressableInAConfigFile(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("genspec", flag.ContinueOnError)
	fs.SetOutput(nil)
	_ = NewWithFlags(fs, os.Stdout, os.Stderr)

	schema, err := configSchema()
	require.NoError(t, err)

	fs.VisitAll(func(f *flag.Flag) {
		if reason, excused := notConfigurable[f.Name]; excused {
			assert.NotContainsf(t, schema, f.Name,
				"-%s is excused from configuration (%s) but the schema addresses it anyway", f.Name, reason)

			return
		}

		assert.Containsf(t, schema, f.Name,
			"flag -%s is addressed in no configuration section. Add it to commandSections, or excuse "+
				"it in notConfigurable with a reason.", f.Name)
	})
}

// read is the koanf half. What it hands back is checked through the flags it presets, end to end; what is worth
// stating here is that a file it cannot use is refused as a bad file, rather than read as an empty one.
func TestReadConfigFile(t *testing.T) {
	t.Parallel()

	t.Run("should read a file into section-qualified keys", func(t *testing.T) {
		// The flat shape cliconf.Parse produces, so that the two readers are interchangeable and the commands that
		// cannot take koanf lose nothing.
		t.Parallel()

		values, err := read(configFile(t, "document:\n  compact: true\nemit:\n  scan-models: false\n"))
		require.NoError(t, err)

		assert.Equal(t, true, values["document"+cliconf.Delimiter+"compact"])
		assert.Equal(t, false, values["emit"+cliconf.Delimiter+"scan-models"])
	})

	t.Run("should refuse a file it cannot read", func(t *testing.T) {
		t.Parallel()

		_, err := read(t.TempDir())

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
	})

	t.Run("should refuse a file it cannot parse", func(t *testing.T) {
		t.Parallel()

		_, err := read(configFile(t, "document:\n\tcompact: true\n")) // tabs do not indent YAML

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
	})
}

// configured is what stands between the file and the flags. Whether the keys are obeyed is checked end to end; what
// it owes on its own is to name the file when it gives up on one.
func TestConfigured(t *testing.T) {
	t.Parallel()

	path := configFile(t, "document:\n\tcompact: true\n") // tabs do not indent YAML
	cfg := parseInto(t, "-config", path)

	_, named, err := configured(cfg.set, cfg.configFile)

	require.ErrorIs(t, err, cliconf.ErrBadConfig)
	assert.Equal(t, path, named, "the file it was reading, which a caller that only searched for it cannot know")
}

func configFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), cliconf.Names[0])
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestConfigSchemaCoversBothHalves(t *testing.T) {
	t.Parallel()

	schema, err := configSchema()
	require.NoError(t, err)

	assert.Equal(t, "scan", schema["workdir"], "the library's own flags")
	assert.Equal(t, "emit", schema["scan-models"])
	assert.Equal(t, sectionDocument, schema["output"], "and this command's")
	assert.Equal(t, sectionDiagnostics, schema["fail-on"])
	assert.Equal(t,
		[]string{sectionDiagnostics, sectionDocument, "emit", "go", "load", "scan"},
		schema.Sections())
}
