// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"

	"github.com/go-openapi/spec"
)

type operationsBuilder struct {
	ctx        *scanCtx
	path       parsedPathContent
	operations map[string]*spec.Operation
}

func (o *operationsBuilder) Build(tgt *spec.Paths) error {
	pthObj := tgt.Paths[o.path.Path]

	op := setPathOperation(
		o.path.Method, o.path.ID,
		&pthObj, o.operations[o.path.ID])

	op.Tags = o.path.Tags

	sp := new(yamlSpecScanner)
	sp.setTitle = func(lines []string) { op.Summary = joinDropLast(lines) }
	sp.setDescription = func(lines []string) { op.Description = joinDropLast(lines) }

	if err := sp.Parse(o.path.Remaining); err != nil {
		return fmt.Errorf("operation (%s): %w", op.ID, err)
	}
	if err := sp.UnmarshalSpec(op.UnmarshalJSON); err != nil {
		return fmt.Errorf("operation (%s): %w", op.ID, err)
	}

	if tgt.Paths == nil {
		tgt.Paths = make(map[string]spec.PathItem)
	}

	tgt.Paths[o.path.Path] = pthObj
	return nil
}

type parsedPathContent struct {
	Method, Path, ID string
	Tags             []string
	Remaining        *ast.CommentGroup
}

func parsePathAnnotation(annotation *regexp.Regexp, lines []*ast.Comment) (cnt parsedPathContent) {
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
				cc.Text = line
				cnt.Remaining.List = append(cnt.Remaining.List, cc)
				justMatched = false
			}
		}
	}

	return cnt
}

// assignOrReuse either reuses an existing operation (if the ID matches)
// or assigns op to the slot.
func assignOrReuse(slot **spec.Operation, op *spec.Operation, id string) *spec.Operation {
	if *slot != nil && id == (*slot).ID {
		return *slot
	}
	*slot = op
	return op
}

func setPathOperation(method, id string, pthObj *spec.PathItem, op *spec.Operation) *spec.Operation {
	if op == nil {
		op = new(spec.Operation)
		op.ID = id
	}

	switch strings.ToUpper(method) {
	case "GET":
		op = assignOrReuse(&pthObj.Get, op, id)
	case "POST":
		op = assignOrReuse(&pthObj.Post, op, id)
	case "PUT":
		op = assignOrReuse(&pthObj.Put, op, id)
	case "PATCH":
		op = assignOrReuse(&pthObj.Patch, op, id)
	case "HEAD":
		op = assignOrReuse(&pthObj.Head, op, id)
	case "DELETE":
		op = assignOrReuse(&pthObj.Delete, op, id)
	case "OPTIONS":
		op = assignOrReuse(&pthObj.Options, op, id)
	}

	return op
}
