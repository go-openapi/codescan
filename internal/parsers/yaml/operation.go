// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"fmt"
	"strings"

	"github.com/go-openapi/swag/yamlutils"
	"go.yaml.in/yaml/v3"
)

// UnmarshalBody runs a raw godoc-comment YAML body through the
// standard godoc → YAML → JSON pipeline expected by every Swagger
// target that consumes JSON-shape input:
//
//  1. RemoveIndent — strip the common indent godoc adds to every
//     line and turn leading tabs into two-space sequences (YAML
//     refuses tab indentation).
//  2. yaml.Unmarshal into a generic map[any]any.
//  3. yamlutils.YAMLToJSON — coerce the map[any]any soup into
//     JSON-shaped values with concrete leaf types.
//  4. Hand the resulting JSON bytes to the caller's callback,
//     typically a *spec.<Target>.UnmarshalJSON or a json.Unmarshal
//     into a caller-provided struct.
//
// Empty body returns nil — the caller's target is left untouched.
//
// Used by the operations bridge (swagger:operation YAML body),
// the meta bridge (securityDefinitions, infoExtensions,
// extensions raw blocks), and any future target that needs the
// same shape.
func UnmarshalBody(body string, unmarshal func([]byte) error) error {
	if body == "" {
		return nil
	}

	lines := strings.Split(body, "\n")
	lines = RemoveIndent(lines)
	normalised := strings.Join(lines, "\n")

	yamlValue := make(map[any]any)
	if err := yaml.Unmarshal([]byte(normalised), &yamlValue); err != nil {
		return fmt.Errorf("yaml body: %w", err)
	}

	jsonValue, err := yamlutils.YAMLToJSON(yamlValue)
	if err != nil {
		return fmt.Errorf("yaml→json: %w", err)
	}

	return unmarshal(jsonValue)
}
