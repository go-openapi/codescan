// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"flag"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Schema says which section of a configuration file each flag is addressed in.
//
// It is what makes a key checkable: without it every key would have to be believed, and a misspelled one
// would read as a setting that quietly never applied.
// Commands build theirs by merging what the shared flag tables declare with what they add themselves.
type Schema map[string]string

// Merge returns s with more added, leaving both alone.
//
// The command's own flags arrive this way, which is also where a collision would show up: a flag
// cannot be addressed in two sections, so the later one silently winning is not a thing to allow.
func (s Schema) Merge(more Schema) (Schema, error) {
	merged := make(Schema, len(s)+len(more))
	maps.Copy(merged, s)

	for name, section := range more {
		if existing, clash := merged[name]; clash {
			return nil, fmt.Errorf("%w: flag %q is declared in sections %q and %q",
				ErrBadConfig, name, existing, section)
		}
		merged[name] = section
	}

	return merged, nil
}

// Sections reports the sections the schema addresses, sorted.
func (s Schema) Sections() []string {
	seen := make(map[string]bool, len(s))
	for _, section := range s {
		seen[section] = true
	}

	return slices.Sorted(maps.Keys(seen))
}

// Result reports what a configuration file did.
type Result struct {
	// Set names the flags the file decided, in order. A flag absent from it was either typed on
	// the command line or left at its default.
	Set []string

	// Ignored names the keys in sections this command does not know - another command's half of a
	// shared file, or a section that was misspelled. Reported rather than dropped so that the second
	// case is findable at all; a caller that mentions them under a -verbose gives that a way out.
	Ignored []string
}

// Apply writes a configuration file's values onto a flag set.
//
// Flags the caller typed are left alone. Everything else goes through [flag.FlagSet.Set], so the
// file is parsed by the same code as the command line and cannot mean anything an argument could not.
func Apply(fs *flag.FlagSet, values map[string]any, schema Schema) (Result, error) {
	var result Result

	known := make(map[string]bool, len(schema))
	for _, section := range schema {
		known[section] = true
	}

	typed := passed(fs)

	// Sorted, so that what is reported and which key an error names do not depend on a map's whim.
	for _, key := range slices.Sorted(maps.Keys(values)) {
		section, name := Split(key)

		switch {
		case section == "":
			return result, fmt.Errorf("%w: %q is in no section; expected one of %s",
				ErrUnknownKey, name, strings.Join(schema.Sections(), ", "))
		case !known[section]:
			result.Ignored = append(result.Ignored, key)

			continue
		}

		if err := check(name, section, schema, fs); err != nil {
			return result, err
		}

		if typed[name] {
			// Typed on the command line, which outranks the file.
			continue
		}

		text, err := stringify(values[key])
		if err != nil {
			return result, fmt.Errorf("%w: %s: %w", ErrBadValue, key, err)
		}
		if err := fs.Set(name, text); err != nil {
			return result, fmt.Errorf("%w: %s: %w", ErrBadValue, key, err)
		}

		result.Set = append(result.Set, name)
	}

	return result, nil
}

// check reports whether a key names a flag this command has, in the section it belongs to.
func check(name, section string, schema Schema, fs *flag.FlagSet) error {
	want, declared := schema[name]
	switch {
	case !declared:
		return fmt.Errorf("%w: %s%s%s", ErrUnknownKey, section, Delimiter, name)
	case want != section:
		return fmt.Errorf("%w: %s%s%s is addressed in section %q, not %q",
			ErrUnknownKey, section, Delimiter, name, want, section)
	}

	if fs.Lookup(name) == nil {
		// The schema promised a flag the command never registered. A caller's mistake rather than the
		// file's, so it says so in those terms.
		return fmt.Errorf("%w: the schema declares %q, which is registered on no flag", ErrBadConfig, name)
	}

	return nil
}

// passed reports which flags appeared on the command line.
//
// Asked once, before anything is written: [flag.FlagSet.Set] records what it sets in the same place,
// so consulting it as we go would let the first value applied make the second look typed.
func passed(fs *flag.FlagSet) map[string]bool {
	typed := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		typed[f.Name] = true
	})

	return typed
}

// stringify renders a configuration value as the flag package would have received it.
//
// Everything is text in the end, because that is what a command line is. A list becomes the
// comma-separated form the list flags already take, so writing one as a YAML sequence and writing it
// as a string are the same thing.
func stringify(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		// An explicitly empty value: `name-from-tags:` with nothing after it. That is a statement, and
		// for the flags that tell an empty list from an absent one it is the interesting one.
		return "", nil
	case string:
		return typed, nil
	case bool:
		return strconv.FormatBool(typed), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64), nil
	case []any:
		return stringifyList(typed)
	default:
		return "", fmt.Errorf("%w: %T is not a value a flag can take", ErrBadValue, value)
	}
}

func stringifyList(values []any) (string, error) {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		part, err := stringify(value)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, ","), nil
}
