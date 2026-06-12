// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package panels holds the three scrollable sub-panels of the genspec-tui
// layout: the source tree (left), the generated spec (right) and the
// diagnostics (bottom).
package panels

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// node is a file or directory in the source tree. Directories list only the
// descendants that (transitively) contain Go files; everything else is pruned
// at build time.
type node struct {
	name     string
	path     string
	isDir    bool
	expanded bool
	depth    int
	children []*node
}

// Tree is the left-hand source-tree explorer. It owns a cursor and a scroll
// offset (the git-janitor Base idiom) and renders a flattened view of the
// expanded nodes inside a bordered, titled box.
type Tree struct {
	root   *node
	flat   []*node // visible rows, recomputed on expand/collapse
	cursor int
	offset int
	w, h   int
}

// NewTree builds the explorer rooted at root, pruned to Go-bearing paths.
func NewTree(root string) Tree {
	t := Tree{root: buildTree(root)}
	t.rebuild()
	return t
}

// SetSize fits the panel to outer dimensions w×h (border + title reserved).
func (p *Tree) SetSize(w, h int) {
	p.w, p.h = w, h
	p.clampOffset()
}

// Selection returns the path of the node under the cursor, whether it is a
// directory, and false when the tree is empty. Used by the model to react to
// the user's current focus (locating diagnostics, opening a file for edit).
func (p *Tree) Selection() (path string, isDir bool, ok bool) {
	n := p.current()
	if n == nil {
		return "", false, false
	}
	return n.path, n.isDir, true
}

// Content returns the flattened tree as indented text, for clipboard copy.
func (p *Tree) Content() string {
	var b strings.Builder
	for i, n := range p.flat {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.Repeat("  ", n.depth))
		b.WriteString(n.name)
		if n.isDir {
			b.WriteString("/")
		}
	}
	return b.String()
}

// Update handles cursor movement and expand/collapse.
func (p *Tree) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch key.MsgBinding(km) {
	case key.Up, key.K:
		p.move(-1)
	case key.Down, key.J:
		p.move(1)
	case key.Right, key.L:
		if n := p.current(); n != nil && n.isDir && !n.expanded {
			n.expanded = true
			p.rebuild()
		}
	case key.Left, key.H:
		p.collapseOrParent()
	case key.Enter:
		if n := p.current(); n != nil && n.isDir {
			n.expanded = !n.expanded
			p.rebuild()
		}
	}
	return nil
}

// View renders the bordered panel; focused brightens the border/title and
// shows the cursor highlight.
func (p *Tree) View(focused bool) string {
	title := theme.Title(focused).Render("source")
	inner := max(p.w-2, 0)
	visible := max(p.h-3, 0)

	var b strings.Builder
	if len(p.flat) <= 1 && (p.root == nil || len(p.root.children) == 0) {
		b.WriteString(theme.Status().Render(fit("(no .go files under root)", inner)))
	} else {
		end := min(p.offset+visible, len(p.flat))
		for i := p.offset; i < end; i++ {
			row := p.renderRow(p.flat[i], inner)
			switch {
			case i == p.cursor && focused:
				row = theme.Selected().Render(row)
			case p.flat[i].isDir:
				row = theme.Dir().Render(row)
			}
			b.WriteString(row)
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	return theme.Panel(p.w, p.h, focused).Render(title + "\n" + b.String())
}

func (p *Tree) renderRow(n *node, width int) string {
	marker := "  "
	if n.isDir {
		if n.expanded {
			marker = "▾ "
		} else {
			marker = "▸ "
		}
	}

	label := strings.Repeat("  ", n.depth) + marker + n.name
	if n.isDir {
		label += "/"
	}
	return fit(label, width)
}

func (p *Tree) move(d int) {
	if len(p.flat) == 0 {
		return
	}
	p.cursor = clamp(p.cursor+d, 0, len(p.flat)-1)
	p.clampOffset()
}

// ScrollBy moves the cursor by delta rows (used for mouse-wheel scrolling).
func (p *Tree) ScrollBy(delta int) { p.move(delta) }

// collapseOrParent collapses an expanded directory, otherwise jumps to the
// parent row.
func (p *Tree) collapseOrParent() {
	n := p.current()
	if n == nil {
		return
	}
	if n.isDir && n.expanded {
		n.expanded = false
		p.rebuild()
		return
	}
	for i := p.cursor - 1; i >= 0; i-- {
		if p.flat[i].depth == n.depth-1 {
			p.cursor = i
			p.clampOffset()
			return
		}
	}
}

func (p *Tree) current() *node {
	if p.cursor < 0 || p.cursor >= len(p.flat) {
		return nil
	}
	return p.flat[p.cursor]
}

// rebuild recomputes the flattened visible-row slice and re-clamps the cursor.
func (p *Tree) rebuild() {
	p.flat = p.flat[:0]
	if p.root != nil {
		flatten(p.root, &p.flat)
	}
	p.cursor = clamp(p.cursor, 0, max(len(p.flat)-1, 0))
	p.clampOffset()
}

func (p *Tree) clampOffset() {
	visible := max(p.h-3, 1)
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+visible {
		p.offset = p.cursor - visible + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

// buildTree walks root, returning its node with Go-bearing descendants
// populated. The root node is always returned (expanded) even when empty.
func buildTree(root string) *node {
	rn := &node{name: filepath.Base(root), path: root, isDir: true, expanded: true}
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		rn.children = readDir(root, 1)
	}
	return rn
}

// readDir returns the directory's child nodes: subdirectories that contain Go
// files (transitively) and *.go files, directories first, each sorted by name.
func readDir(dir string, depth int) []*node {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var dirs, files []*node
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		if e.IsDir() {
			if name == "vendor" || name == "node_modules" {
				continue
			}
			child := &node{name: name, path: filepath.Join(dir, name), isDir: true, depth: depth}
			child.children = readDir(child.path, depth+1)
			if len(child.children) > 0 { // prune dirs with no Go content
				dirs = append(dirs, child)
			}
			continue
		}

		if strings.HasSuffix(name, ".go") {
			files = append(files, &node{name: name, path: filepath.Join(dir, name), depth: depth})
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return append(dirs, files...)
}

func flatten(n *node, out *[]*node) {
	*out = append(*out, n)
	if n.isDir && n.expanded {
		for _, c := range n.children {
			flatten(c, out)
		}
	}
}

// fit truncates s to width with an ellipsis, or right-pads it with spaces so
// the cursor highlight spans the full inner width.
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > width {
		if width == 1 {
			return "…"
		}
		return string(r[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(r))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
