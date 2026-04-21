// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package grammartest holds the parity harness the P5 builder
// migration depends on. Two pieces live here:
//
//   - NormalizedCommentView is a parser-agnostic description of what
//     a single comment group says. Both the legacy regex parser (via
//     the TypeIndex scan path) and the v2 grammar parser lift their
//     interpretation into this view so the harness can diff at a
//     common level instead of comparing interpreted spec.* output.
//
//   - AssertGoldenView drives the harness: parse, normalise, and
//     compare against a committed JSON snapshot under
//     testdata/golden/. UPDATE_GOLDEN=1 rewrites snapshots when the
//     v2 parser deliberately changes output.
//
// Pre-P5 the harness exercises only the v2 path (there is no v2
// builder to exit through yet). The v1 adapter lands when P5
// bridge-taggers do, giving us the old-vs-new diff the plan calls
// for. Treating the harness as "v2 regression suite first, full
// parity runner second" keeps it useful without blocking on P5.
package grammartest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// NormalizedCommentView is the diff-friendly representation of a
// single parsed comment group. Fields are JSON-serialisable and
// sorted where order is not meaningful (extensions, YAML bodies are
// kept in source order).
type NormalizedCommentView struct {
	// AnnotationKind is the top-level swagger:<name> (or "unknown"
	// for unbound comments).
	AnnotationKind string `json:"annotationKind"`

	// AnnotationArgs holds the kind-specific positional arguments
	// resolved by the parser — empty for annotations with none.
	AnnotationArgs AnnotationArgs `json:"annotationArgs,omitzero"`

	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	Properties []NormalizedProperty  `json:"properties,omitempty"`
	YAMLBlocks []NormalizedYAMLBlock `json:"yamlBlocks,omitempty"`
	Extensions []NormalizedExtension `json:"extensions,omitempty"`

	// Diagnostics are sorted by Code to make diffs deterministic
	// (severity, message text may vary).
	Diagnostics []NormalizedDiagnostic `json:"diagnostics,omitempty"`
}

// AnnotationArgs captures the per-kind positional data the parser
// extracts. Only fields relevant to the emitted block's kind are
// populated; others stay zero and are omitted from JSON.
type AnnotationArgs struct {
	Name        string   `json:"name,omitempty"`
	Method      string   `json:"method,omitempty"`
	Path        string   `json:"path,omitempty"`
	Tags        string   `json:"tags,omitempty"`
	OpID        string   `json:"opId,omitempty"`
	TargetTypes []string `json:"targetTypes,omitempty"`
}

// NormalizedProperty is a Block Property in a diff-stable form. Value
// and Typed are both captured — raw string for eye-readability in the
// golden file, typed forms for exact assertions.
type NormalizedProperty struct {
	Keyword    string   `json:"keyword"`
	Value      string   `json:"value,omitempty"`
	ItemsDepth int      `json:"itemsDepth,omitempty"`
	Body       []string `json:"body,omitempty"`

	// Typed is populated only for parse-time-convertible ValueTypes
	// (Number, Integer, Boolean, StringEnum); otherwise nil.
	Typed *NormalizedTyped `json:"typed,omitempty"`
}

// NormalizedTyped is the typed-value subtree. Only one of the
// scalar fields is populated per the Type label.
type NormalizedTyped struct {
	Type string `json:"type"` // "number" | "integer" | "boolean" | "string-enum"

	Op      string  `json:"op,omitempty"` // only for number
	Number  float64 `json:"number,omitempty"`
	Integer int64   `json:"integer,omitempty"`
	Boolean bool    `json:"boolean,omitempty"`
	String  string  `json:"string,omitempty"`
}

// NormalizedYAMLBlock holds one captured --- fence body.
type NormalizedYAMLBlock struct {
	Text string `json:"text"`
}

// NormalizedExtension is one x-* / non-x- entry from an extensions
// block.
type NormalizedExtension struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// NormalizedDiagnostic captures the Code + Severity of a diagnostic.
// Position/Message are intentionally elided — message wording may
// change without a real regression; position noise can dominate the
// diff. The code is the stable contract.
type NormalizedDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
}

// ViewFromBlock converts a grammar.Block into its normalized view.
// Stable for diffing: the function is pure, produces deterministic
// output for identical inputs, and sorts Diagnostics by (code,
// severity) so transient ordering can't flake tests.
func ViewFromBlock(b grammar.Block) NormalizedCommentView {
	v := NormalizedCommentView{
		AnnotationKind: b.AnnotationKind().String(),
		Title:          b.Title(),
		Description:    b.Description(),
	}

	addAnnotationArgs(&v, b)

	for p := range b.Properties() {
		v.Properties = append(v.Properties, normalizeProperty(p))
	}
	for y := range b.YAMLBlocks() {
		v.YAMLBlocks = append(v.YAMLBlocks, NormalizedYAMLBlock{Text: y.Text})
	}
	for e := range b.Extensions() {
		v.Extensions = append(v.Extensions, NormalizedExtension{Name: e.Name, Value: e.Value})
	}
	v.Diagnostics = normalizeDiagnostics(b.Diagnostics())

	return v
}

// addAnnotationArgs fills AnnotationArgs from the concrete Block
// kind, via a type-switch that's exhaustive over the current family.
func addAnnotationArgs(v *NormalizedCommentView, b grammar.Block) {
	switch tb := b.(type) {
	case *grammar.ModelBlock:
		v.AnnotationArgs.Name = tb.Name
	case *grammar.ResponseBlock:
		v.AnnotationArgs.Name = tb.Name
	case *grammar.ParametersBlock:
		v.AnnotationArgs.TargetTypes = append([]string(nil), tb.TargetTypes...)
	case *grammar.RouteBlock:
		v.AnnotationArgs.Method = tb.Method
		v.AnnotationArgs.Path = tb.Path
		v.AnnotationArgs.Tags = tb.Tags
		v.AnnotationArgs.OpID = tb.OpID
	case *grammar.OperationBlock:
		v.AnnotationArgs.Method = tb.Method
		v.AnnotationArgs.Path = tb.Path
		v.AnnotationArgs.Tags = tb.Tags
		v.AnnotationArgs.OpID = tb.OpID
	case *grammar.MetaBlock, *grammar.UnboundBlock:
		// No kind-specific positional args.
	}
}

func normalizeProperty(p grammar.Property) NormalizedProperty {
	np := NormalizedProperty{
		Keyword:    p.Keyword.Name,
		Value:      p.Value,
		ItemsDepth: p.ItemsDepth,
	}
	if len(p.Body) > 0 {
		np.Body = append([]string(nil), p.Body...)
	}
	if p.Typed.Type != grammar.ValueNone {
		np.Typed = &NormalizedTyped{
			Type:    p.Typed.Type.String(),
			Op:      p.Typed.Op,
			Number:  p.Typed.Number,
			Integer: p.Typed.Integer,
			Boolean: p.Typed.Boolean,
			String:  p.Typed.String,
		}
	}
	return np
}

func normalizeDiagnostics(diags []grammar.Diagnostic) []NormalizedDiagnostic {
	if len(diags) == 0 {
		return nil
	}
	out := make([]NormalizedDiagnostic, len(diags))
	for i, d := range diags {
		out[i] = NormalizedDiagnostic{
			Code:     string(d.Code),
			Severity: d.Severity.String(),
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Severity < out[j].Severity
	})
	return out
}

// ParseSourceToViews runs the v2 grammar parser over every comment
// group attached to a declaration in the given Go source snippet,
// returning one NormalizedCommentView per group. Deterministic
// ordering: source order of the declarations, which go/parser
// preserves.
func ParseSourceToViews(t *testing.T, src string) []NormalizedCommentView {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "t.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	p := grammar.NewParser(fset)

	var views []NormalizedCommentView
	for _, decl := range f.Decls {
		cg := docOf(decl)
		if cg == nil {
			continue
		}
		views = append(views, ViewFromBlock(p.Parse(cg)))
	}
	return views
}

func docOf(decl ast.Decl) *ast.CommentGroup {
	switch d := decl.(type) {
	case *ast.GenDecl:
		return d.Doc
	case *ast.FuncDecl:
		return d.Doc
	}
	return nil
}

// AssertGoldenView compares the given views to the JSON snapshot at
// path. When the env var UPDATE_GOLDEN=1 is set, the snapshot is
// rewritten (matching the project's existing scantest convention).
// Otherwise an unexpected mismatch fails the test with a diff-ready
// error.
func AssertGoldenView(t *testing.T, path string, views []NormalizedCommentView) {
	t.Helper()
	got, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatalf("marshal views: %v", err)
	}
	// Trailing newline so editors don't complain.
	got = append(got, '\n')

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		// Golden files are committed to git and read by humans/CI;
		// standard perms, no secrets.
		const (
			dirMode  = 0o755
			fileMode = 0o644
		)
		if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(path, got, fileMode); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\n(run with UPDATE_GOLDEN=1 to create)", path, err)
	}
	// Normalise line endings: git on Windows checks files out with
	// CRLF by default (core.autocrlf=true), which would otherwise
	// break byte-equal comparison against the LF-only output of
	// json.MarshalIndent. Normalising on the read side only is
	// correct because `got` is always freshly LF-produced.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(got, want) {
		t.Fatalf(
			"golden mismatch at %s\n--- want (%d bytes)\n%s\n--- got (%d bytes)\n%s\n(run with UPDATE_GOLDEN=1 to accept)",
			path, len(want), want, len(got), got,
		)
	}
	_ = fmt.Sprintf // keep fmt used if trimmed
}
