// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-openapi/codescan/internal/parsers/yaml"
)

const (
	// kvParts is the number of parts when splitting key:value pairs.
	kvParts = 2
)

// Many thanks go to https://github.com/yvasiyarov/swagger
// this is loosely based on that implementation but for swagger 2.0

type ConsumesDropEmptyParser struct {
	*multilineYAMLListParser
}

func NewConsumesDropEmptyParser(set func([]string)) *ConsumesDropEmptyParser {
	return &ConsumesDropEmptyParser{
		multilineYAMLListParser: &multilineYAMLListParser{
			set: set,
			rx:  rxConsumes,
		},
	}
}

type ProducesDropEmptyParser struct {
	*multilineYAMLListParser
}

func NewProducesDropEmptyParser(set func([]string)) *ProducesDropEmptyParser {
	return &ProducesDropEmptyParser{
		multilineYAMLListParser: &multilineYAMLListParser{
			set: set,
			rx:  rxProduces,
		},
	}
}

// multilineYAMLListParser is the Q4 replacement for
// multilineDropEmptyParser on list-valued block bodies
// (`consumes:` / `produces:` in meta + operation scope). The
// body is captured raw — its list-item markers (`- value`)
// survive the preprocessor — and interpreted by
// internal/parsers/yaml/ as a YAML list. Strict list: non-list
// bodies emit a warning and produce no values.
//
// See `.claude/plans/workshops/w2-enum.md` §2.6 (quirk 4),
// `grammar-parser-architecture.md` §3.3 (sub-parser pattern),
// and `.claude/plans/forthcoming-features.md` §5.2 (P7.7 doc
// follow-up).
type multilineYAMLListParser struct {
	set func([]string)
	rx  *regexp.Regexp
}

func (m *multilineYAMLListParser) Matches(line string) bool {
	return m.rx.MatchString(line)
}

func (m *multilineYAMLListParser) Parse(lines []string) error {
	// Strip comment noise but preserve `-` (the YAML list marker).
	// Matches rxUncommentHeaders minus the dash.
	cleaned := cleanupScannerLines(lines, rxUncommentNoDash)

	// Drop drop-empty to avoid blank spacer lines between items
	// confusing the YAML parser.
	cleaned = removeEmptyLines(cleaned)
	if len(cleaned) == 0 {
		return nil
	}

	body := strings.Join(cleaned, "\n")
	parsed, err := yaml.Parse(body)
	if err != nil {
		log.Printf("WARNING: parse.invalid-block-body: %v", err)
		return nil
	}
	list, ok := parsed.([]any)
	if !ok {
		log.Printf("WARNING: parse.invalid-block-body: expected YAML list, got %T", parsed)
		return nil
	}

	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, fmt.Sprintf("%v", item))
	}
	m.set(out)
	return nil
}
