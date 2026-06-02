// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"go/ast"
	"regexp"
	"strings"
)

var (
	rxStripComments = regexp.MustCompile(`^[^\p{L}\p{N}\p{Pd}\p{Pc}\+]*`)
	rxSpace         = regexp.MustCompile(`\p{Zs}+`)
)

type ParsedPathContent struct {
	Method, Path, ID string
	Tags             []string
	Remaining        *ast.CommentGroup
}

func ParseOperationPathAnnotation(lines []*ast.Comment) (cnt ParsedPathContent) {
	return parsePathAnnotation(rxOperation, lines)
}

func ParseRoutePathAnnotation(lines []*ast.Comment) (cnt ParsedPathContent) {
	return parsePathAnnotation(rxRoute, lines)
}

// ensureCommentMarker returns line with a leading `// ` prepended
// unless it already starts with `//` or `/*`. Used by
// parsePathAnnotation when reshaping a multi-line block comment into
// per-line *ast.Comment entries; the grammar lexer only runs its
// content-prefix strip on the `//` / `/*` branches of stripComment.
func ensureCommentMarker(line string) string {
	if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
		return line
	}
	return "// " + line
}

func parsePathAnnotation(annotation *regexp.Regexp, lines []*ast.Comment) (cnt ParsedPathContent) {
	const routeTagsIndex = 3 // routeTagsIndex is the regex submatch index where route tags begin.
	var justMatched bool

	for _, cmt := range lines {
		txt := cmt.Text
		for line := range strings.SplitSeq(txt, "\n") {
			matches := annotation.FindStringSubmatch(line)
			if len(matches) > routeTagsIndex {
				cnt.Method, cnt.Path, cnt.ID = matches[1], matches[2], matches[len(matches)-1]
				cnt.Tags = rxSpace.Split(matches[3], -1)
				if len(matches[3]) == 0 {
					cnt.Tags = nil
				}
				justMatched = true

				continue
			}

			if cnt.Method == "" {
				continue
			}

			if cnt.Remaining == nil {
				cnt.Remaining = new(ast.CommentGroup)
			}

			if !justMatched || strings.TrimSpace(rxStripComments.ReplaceAllString(line, "")) != "" {
				cc := new(ast.Comment)
				cc.Slash = cmt.Slash
				// Force a `//` prefix on the synthetic per-line
				// comment so grammar's lexer sees a shape it strips
				// (the `//` branch of stripComment runs
				// trimContentPrefix, which sheds leading ` \t*/|`).
				// Without the prefix, the lexer falls through its
				// default case and preserves the source line
				// verbatim — leading tabs from `/* ... */` block-
				// comment route docs then leak into Title /
				// Description. Lines that already start with `//`
				// or `/*` are left alone so their leading whitespace
				// is recorded correctly via the matching strip path.
				cc.Text = ensureCommentMarker(line)
				cnt.Remaining.List = append(cnt.Remaining.List, cc)
				justMatched = false
			}
		}
	}

	return cnt
}
