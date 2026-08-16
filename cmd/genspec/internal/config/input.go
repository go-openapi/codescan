// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
)

// loadInputSpec reads the specification the scan's discoveries are merged into.
//
// What the scan finds is written on top of this document rather than beside it, so it is the place
// for everything a scanner cannot know: the host, the security definitions, a hand-written path the
// annotations do not describe.
func loadInputSpec(path string) (*spec.Swagger, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read -input: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%w: -input %q is a directory, not a specification", sentinel.ErrUsage, path)
	}

	document, err := loads.Spec(path)
	if err != nil {
		return nil, fmt.Errorf("cannot load -input %q: %w", path, err)
	}

	return document.Spec(), nil
}
