// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Delimiter joins a section to the key inside it.
const Delimiter = "."

// YAML reads a configuration file.
//
// The methods are shaped as koanf's parser interface, which it declares structurally, so a command
// using koanf can hand this straight to it - and this package still owes koanf no import. JSON needs
// no parser of its own: it is a subset of YAML, so a file written as JSON reads here unchanged.
type YAML struct{}

// Unmarshal reads a configuration file into the nested map it describes.
func (YAML) Unmarshal(data []byte) (map[string]any, error) {
	var nested map[string]any
	if err := yaml.Unmarshal(data, &nested); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadConfig, err)
	}

	if nested == nil {
		// An empty file, or one holding nothing but comments. Not an error - it says nothing, which
		// is a thing a configuration file is allowed to say.
		return map[string]any{}, nil
	}

	return nested, nil
}

// Marshal writes what Unmarshal reads. Present to complete koanf's parser interface.
func (YAML) Marshal(values map[string]any) ([]byte, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadConfig, err)
	}

	return data, nil
}

// Parse reads a configuration file into the flat, section-qualified keys [Apply] takes.
//
// This is the whole of the dependency-free path: a command that does not want koanf calls this, and
// gets exactly what koanf's own All() would have handed it.
func Parse(data []byte) (map[string]any, error) {
	nested, err := YAML{}.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return Flatten(nested), nil
}

// Flatten renders nested sections as section-qualified keys.
//
// Only maps are descended into. A list is a value - it is how a repeated flag is written - and
// flattening one would turn the entries of exclude-tags into keys named after their positions.
func Flatten(nested map[string]any) map[string]any {
	flat := make(map[string]any, len(nested))
	flatten("", nested, flat)

	return flat
}

func flatten(prefix string, nested, flat map[string]any) {
	for key, value := range nested {
		qualified := key
		if prefix != "" {
			qualified = prefix + Delimiter + key
		}

		if section, ok := value.(map[string]any); ok {
			flatten(qualified, section, flat)

			continue
		}

		flat[qualified] = value
	}
}

// Split separates a key into the section it is in and the flag it names.
//
// A key with no section is reported as one with an empty section, which Apply refuses: sections are
// what let one file serve several commands, so a key outside them addresses nobody in particular.
func Split(key string) (section, name string) {
	section, name, found := strings.Cut(key, Delimiter)
	if !found {
		return "", section
	}

	return section, name
}
