// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/go-openapi/spec"
	"go.yaml.in/yaml/v3"
)

// documentMode is what a written specification is created with.
const documentMode os.FileMode = 0o644

// render marshals the document in the format asked for.
//
// The JSON rendering is produced either way: it is what YAML is derived from, and what -validate
// checks, so a caller reading the document and a caller validating it are looking at the same bytes.
func render(doc *spec.Swagger, format string, compact bool) (rendered, asJSON []byte, err error) {
	asJSON, err = marshalJSON(doc, compact)
	if err != nil {
		return nil, nil, err
	}

	if format != formatYAML {
		return asJSON, asJSON, nil
	}

	rendered, err = marshalYAML(asJSON)
	if err != nil {
		return nil, nil, err
	}

	return rendered, asJSON, nil
}

func marshalJSON(doc *spec.Swagger, compact bool) ([]byte, error) {
	if compact {
		return json.Marshal(doc)
	}

	return json.MarshalIndent(doc, "", "  ")
}

// marshalYAML re-reads the JSON rendering and writes it back out as YAML.
//
// Through a plain any rather than the spec types, because that is the only thing that renders every
// vendor extension the same way the JSON does. The cost is key order: a Go map does not keep one,
// so a YAML document comes out alphabetical where the JSON came out in the order the spec types
// declare. The information is the same; the diff against a hand-written file is not.
func marshalYAML(asJSON []byte) ([]byte, error) {
	var document any
	if err := yaml.Unmarshal(asJSON, &document); err != nil {
		return nil, fmt.Errorf("cannot re-read the document as YAML: %w", err)
	}

	return yaml.Marshal(document)
}

// write puts the rendered document where -output says.
func write(rendered []byte, output string, stdout io.Writer) error {
	if output == "" || output == "-" {
		_, err := fmt.Fprintf(stdout, "%s\n", rendered)

		return err
	}

	if err := os.WriteFile(output, append(rendered, '\n'), documentMode); err != nil {
		return fmt.Errorf("cannot write %s: %w", output, err)
	}

	return nil
}
