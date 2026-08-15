// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"os"
	"testing"

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
