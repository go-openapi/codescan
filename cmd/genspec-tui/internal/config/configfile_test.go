// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The schema is the whole of what a file may address here, and a shared file is read by more than one command - so
// these are promises to whoever writes the file, not implementation details.
func TestConfigSchema(t *testing.T) {
	t.Parallel()

	t.Run("should address every flag in a section", func(t *testing.T) {
		// A flag with no section can be typed but not configured, and nothing would notice: from the outside that
		// reads exactly like the option not existing.
		t.Parallel()

		cli := newTestFlags(t)

		schema, err := configSchema()
		require.NoError(t, err)

		cli.set.VisitAll(func(f *flag.Flag) {
			if reason, excused := notConfigurable[f.Name]; excused {
				assert.NotContainsf(t, schema, f.Name,
					"-%s is excused from configuration (%s) but the schema addresses it anyway", f.Name, reason)

				return
			}

			assert.Containsf(t, schema, f.Name,
				"flag -%s is addressed in no configuration section. Add it to commandSections, or excuse it in "+
					"notConfigurable with a reason.", f.Name)
		})
	})

	t.Run("should cover both halves", func(t *testing.T) {
		t.Parallel()

		schema, err := configSchema()
		require.NoError(t, err)

		assert.Equal(t, "emit", schema["scan-models"], "the library's own flags")
		assert.Equal(t, "load", schema["stub-stdlib"])
		assert.Equal(t, sectionProfile, schema["mem-profile-rate"], "and this command's")

		// The path options are command-line only. A file is found by searching upwards, so it belongs
		// to the tree being scanned, and that tree may not choose where the session reads or writes.
		assert.NotContains(t, schema, "workdir")
		assert.NotContains(t, schema, "profile-dir")
		assert.Equal(t,
			[]string{"emit", "go", "load", sectionProfile, "scan"},
			schema.Sections(),
			"genspec's document and diagnostics sections are not among them: this command writes no document and "+
				"reports no diagnostics to a stream",
		)
	})

	t.Run("should excuse only flags that exist", func(t *testing.T) {
		// An excuse naming a flag that has since been renamed excuses nothing, and would leave the real flag
		// unaddressable with the guard above still passing.
		t.Parallel()

		cli := newTestFlags(t)

		for name, reason := range notConfigurable {
			assert.NotNilf(t, cli.set.Lookup(name),
				"notConfigurable excuses -%s (%s), but there is no such flag", name, reason)
		}
	})
}
