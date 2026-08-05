// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"go/build"
	"io/fs"
	"path"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/go-openapi/codescan/internal/packages/vfs"
)

// Resolver turns patterns and import paths into directories.
//
// This is the half of `go list` we have to own: mapping "./..." or "example.com/x/y" onto a place on
// the filesystem. It is deliberately narrow — main module, workspace members, the module cache, a
// vendor directory, and GOROOT — because those are the trees whose layout is knowable without running
// the go command.
type Resolver struct {
	vfs *vfs.FS
	ctx *build.Context
	dir string
	env map[string]string

	// gowork is the caller's GOWORK setting: "off", a path to a go.work, or "" to search upwards.
	gowork string

	// modFlag is the caller's -mod setting ("mod", "vendor", "readonly" or ""), which decides whether a
	// vendor directory is authoritative.
	modFlag string

	// stubStdlib withholds the standard library from resolution: its imports are synthesized instead.
	stubStdlib bool

	modRoot string // directory holding the main module's go.mod ("" if none found)
	modPath string // the main module's path
	vendor  string // <modRoot>/vendor when it is authoritative (see vendorMode), else ""
	srcRoot string // GOROOT/src

	// modDirs maps a required module path to the directory holding its source: the module cache for
	// an ordinary requirement, an arbitrary directory for a `replace` target.
	//
	// Since Go 1.17 the main module's go.mod lists every relevant dependency, direct and indirect, so
	// reading it is enough to place imports without walking the whole module graph.
	modDirs map[string]string

	// ws is the governing go.work, or nil. Its `use` directives place sibling modules at their working
	// copies, which is the only reason a workspace changes what a scan sees.
	ws *workspace

	// nearestMod memoizes directory -> enclosing module, so deriving an import path does not re-walk
	// the tree once per package.
	nearestMod map[string]moduleAt
}

// moduleAt is the module a directory belongs to.
type moduleAt struct {
	root string // directory holding go.mod ("" if the directory is in no module)
	path string // that module's declared path
}

// maxParentWalk bounds every upward search, so a filesystem loop cannot hang the loader.
const maxParentWalk = 64

// Target is a package the caller asked for: where it is, and what it is called.
type Target struct {
	Dir     string
	PkgPath string
}

// Config is what a Resolver needs to know before it can place anything.
type Config struct {
	// FS is the filesystem every read goes through.
	FS *vfs.FS

	// Context carries the build target and the go/build hooks bound to FS.
	Context *build.Context

	// Dir is the directory patterns are relative to.
	Dir string

	// Env is the effective go environment, for GOMODCACHE and GOPATH.
	Env map[string]string

	// GOWORK selects the workspace: "off", a path, or "" to search upwards.
	GOWORK string

	// ModFlag is the -mod setting, which decides whether a vendor directory is authoritative.
	ModFlag string

	// StubStdlib withholds the standard library from resolution.
	StubStdlib bool
}

// NewResolver builds a Resolver and reads the module context around Config.Dir.
//
// It fails only when go.mod exists and cannot be read: with no requirement placed, every dependency
// would fall through to synthesis and the real fault would vanish behind a wall of warnings.
func NewResolver(cfg Config) (*Resolver, error) {
	r := &Resolver{
		vfs: cfg.FS, ctx: cfg.Context, dir: cfg.Dir, env: cfg.Env,
		gowork: cfg.GOWORK, modFlag: cfg.ModFlag, stubStdlib: cfg.StubStdlib,
	}
	if err := r.init(); err != nil {
		return nil, err
	}

	return r, nil
}

// ResolvePatterns expands the caller's patterns into concrete package directories.
//
// Supported: "./dir", "./dir/...", "dir", "all"-free import paths, and bare "...". Anything the go
// command supports beyond that (query syntax, "std", test patterns) is out of scope.
func (r *Resolver) ResolvePatterns(patterns []string) ([]Target, error) {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	seen := map[string]bool{}
	var out []Target

	for _, pat := range patterns {
		base, recursive := patternRoot(pat)

		dir, pkgPath, ok := r.locate(base)
		if !ok {
			return nil, fmt.Errorf("%w %q relative to %q", ErrUnresolvedPattern, pat, r.dir)
		}

		// A wildcard is matched against the name the caller wrote in: a filesystem pattern against the
		// directory path relative to the walk, an import-path pattern against the import path. Mixing
		// the two would make ./... and example.com/... behave differently for the same tree.
		match := matchPattern(pat)
		relative := isRelativePattern(base)

		if !recursive {
			if !seen[dir] && r.hasGoFiles(dir) {
				seen[dir] = true
				out = append(out, Target{Dir: dir, PkgPath: pkgPath})
			}
			continue
		}

		err := r.vfs.WalkDirs(dir, func(d string) error {
			// `...` stops at a module boundary, as it does for the go command: a nested module is a
			// different module, and its packages are not part of this pattern. Without this the walk
			// swallows vendored trees and sibling modules, and — worse — labels them with import paths
			// derived from the wrong module.
			if d != dir && r.isModuleRoot(d) {
				return fs.SkipDir
			}

			candidate := r.pkgPathFor(d)
			name := candidate
			if relative {
				name = relPatternName(base, dir, d, r.vfs)
			}
			if !seen[d] && match(name) && r.hasGoFiles(d) {
				seen[d] = true
				out = append(out, Target{Dir: d, PkgPath: candidate})
			}

			// A wildcard never reaches INSIDE a vendor directory — "./..." is documented not to match
			// packages under ./vendor or ./mycode/vendor. The directory itself is not special though: a
			// package that merely happens to be called vendor is an ordinary match, which is why this
			// prunes the children rather than the node.
			if d != dir && path.Base(d) == "vendor" {
				return fs.SkipDir
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PkgPath < out[j].PkgPath })
	return out, nil
}

// ResolveImport maps an import path onto a directory.
func (r *Resolver) ResolveImport(importPath string) (dir, pkgPath string, ok bool) {
	// Standard library: GOROOT/src is laid out by import path.
	if !strings.Contains(firstSegment(importPath), ".") {
		if r.stubStdlib {
			return "", "", false
		}
		d := r.vfs.Join(r.srcRoot, importPath)
		if r.vfs.IsDir(d) {
			return d, importPath, true
		}
	}
	// A workspace member, placed by go.work. This comes before the module cache on purpose: the point
	// of a workspace is that a sibling resolves to the copy being worked on, and reading the cached
	// release instead is exactly the staleness `use` exists to prevent.
	if modPath, modDir, found := r.ws.lookup(importPath); found {
		rel := strings.TrimPrefix(strings.TrimPrefix(importPath, modPath), "/")
		if d := r.vfs.Join(modDir, rel); r.vfs.IsDir(d) {
			return d, importPath, true
		}
	}

	// Inside the main module.
	if r.modPath != "" && (importPath == r.modPath || strings.HasPrefix(importPath, r.modPath+"/")) {
		rel := strings.TrimPrefix(strings.TrimPrefix(importPath, r.modPath), "/")
		d := r.vfs.Join(r.modRoot, rel)
		if r.vfs.IsDir(d) {
			return d, importPath, true
		}
	}
	// A vendored dependency. Before the module cache, not after: a vendored build is meant to read what
	// is in the tree, and that is usually the whole reason it was vendored. The cache stays as a
	// fallback for a package vendoring missed, where the go command refuses the build outright — a
	// scanner is more useful degrading than failing.
	if r.vendor != "" {
		if d := r.vfs.Join(r.vendor, importPath); r.vfs.IsDir(d) {
			return d, importPath, true
		}
	}

	// A required module, placed from go.mod.
	for modPath, modDir := range r.modDirs {
		if importPath != modPath && !strings.HasPrefix(importPath, modPath+"/") {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(importPath, modPath), "/")
		if d := r.vfs.Join(modDir, rel); r.vfs.IsDir(d) {
			return d, importPath, true
		}
	}
	return "", "", false
}

// InMainModule reports whether an import path names a package of the module being scanned.
//
// It decides where a package's types may come from. The code under scan must always be read from
// source: its comments are the annotations, and export data carries none. Everything else is a
// dependency, whose types are all that is wanted from it.
func (r *Resolver) InMainModule(importPath string) bool {
	if r.modPath == "" {
		return false
	}

	return importPath == r.modPath || strings.HasPrefix(importPath, r.modPath+"/")
}

func (r *Resolver) init() error {
	if r.dir == "" {
		r.dir = "."
	}
	r.srcRoot = r.vfs.Join(r.ctx.GOROOT, "src")
	r.nearestMod = map[string]moduleAt{}
	r.ws = r.findWorkspace(r.gowork)

	return r.findModule()
}

// findModule walks up from Dir looking for go.mod. Without it, import paths inside the tree under
// scan cannot be mapped to directories at all.
//
// Finding none is not an error: the go command refuses a tree with no module, but a scanner that can
// still read the declarations in front of it is more useful than one that cannot.
func (r *Resolver) findModule() error {
	dir := r.dir
	for range maxParentWalk {
		gomod := r.vfs.Join(dir, "go.mod")
		if blob, err := r.vfs.ReadFile(gomod); err == nil {
			if mp := modfile.ModulePath(blob); mp != "" {
				r.modRoot, r.modPath = dir, mp
				if v := r.vfs.Join(dir, "vendor"); r.vendorMode(v) {
					r.vendor = v
				}

				return r.readRequirements(gomod, blob)
			}
		}
		parent := r.vfs.Join(dir, "..")
		if parent == dir || dir == "." || dir == "/" {
			return nil
		}
		dir = parent
	}

	return nil
}

// vendorMode reports whether dir is a vendor directory the build must actually use.
//
// The go command's test is vendor/modules.txt, not the directory: `go mod vendor` writes that file,
// and without it the tree is not a vendored build — it is a directory that happens to be called
// vendor, and may even be an ordinary package. Reading it anyway is how a stale copy silently
// replaced the real dependency.
//
// -mod=mod is the explicit "ignore the vendor directory" switch, and wins.
func (r *Resolver) vendorMode(dir string) bool {
	if r.modFlag == "mod" {
		return false
	}
	_, err := r.vfs.ReadFile(r.vfs.Join(dir, "modules.txt"))

	return err == nil
}

// readRequirements places every module the main go.mod names, so imports into them resolve.
//
// A vendored tree needs none of this; it matters for the ordinary case where dependencies live in the
// module cache.
func (r *Resolver) readRequirements(gomod string, blob []byte) error {
	// Strict Parse, despite the forward-compatibility cost: ParseLax exists for reading a DEPENDENCY's
	// go.mod and deliberately ignores replace, which is exactly the directive that decides where a
	// dependency is read from. Losing replaces silently would be worse than failing on a go.mod that
	// names a directive this x/mod has not heard of — and that failure is loud and fixed by a bump.
	f, err := modfile.Parse(gomod, blob, nil)
	if err != nil {
		// Degrading here is what makes this worth failing over: with no requirements placed, every
		// dependency falls through to synthesis, and the scan reports a wall of synthesized-import
		// warnings that say nothing about the actual fault.
		return fmt.Errorf("%w: %w", ErrInvalidGoMod, err)
	}

	r.modDirs = make(map[string]string, len(f.Require))
	selected := make(map[string]string, len(f.Require))

	cache := r.moduleCache()
	for _, req := range f.Require {
		selected[req.Mod.Path] = req.Mod.Version
		if cache == "" {
			continue
		}
		if esc, err := module.EscapePath(req.Mod.Path); err == nil {
			r.modDirs[req.Mod.Path] = r.vfs.Join(cache, esc+"@"+req.Mod.Version)
		}
	}

	// `replace` wins over the cache, and a directory target may sit anywhere.
	for _, rep := range f.Replace {
		// A replace pinned to a version applies to that version and no other. Applying it regardless
		// silently swaps in a substitute the build never asked for — and the two forms are easy to
		// confuse, since the unversioned one, which does apply to every version, looks almost the same.
		if v := rep.Old.Version; v != "" && v != selected[rep.Old.Path] {
			continue
		}

		switch {
		case rep.New.Version == "": // filesystem replacement
			dir := rep.New.Path
			if !r.vfs.IsAbs(dir) {
				dir = r.vfs.Join(r.modRoot, dir)
			}
			r.modDirs[rep.Old.Path] = dir
		case cache != "":
			if esc, err := module.EscapePath(rep.New.Path); err == nil {
				r.modDirs[rep.Old.Path] = r.vfs.Join(cache, esc+"@"+rep.New.Version)
			}
		}
	}

	return nil
}

// moduleCache locates the module cache from the environment, falling back to GOPATH/pkg/mod.
func (r *Resolver) moduleCache() string {
	if c := r.env["GOMODCACHE"]; c != "" {
		return c
	}
	gopath := r.env["GOPATH"]
	if gopath == "" {
		gopath = r.ctx.GOPATH
	}
	if gopath == "" {
		return ""
	}
	// GOPATH may be a list; the module cache lives under the first entry.
	if i := strings.IndexAny(gopath, ":;"); i >= 0 {
		gopath = gopath[:i]
	}
	return r.vfs.Join(gopath, "pkg", "mod")
}

// isRelativePattern reports whether a pattern names a place on the filesystem rather than an import
// path.
func isRelativePattern(pat string) bool {
	return pat == "." || pat == ".." ||
		strings.HasPrefix(pat, "./") || strings.HasPrefix(pat, "../") || strings.HasPrefix(pat, "/")
}

// relPatternName renders a walked directory in the same shape as the pattern that is matching it, so
// "./internal/..." can be compared with "./internal/packages".
func relPatternName(base, walkRoot, dir string, v *vfs.FS) string {
	rel, ok := v.HasSubdir(walkRoot, dir)
	if !ok {
		return dir
	}
	if rel == "." {
		return base
	}

	return strings.TrimSuffix(base, "/") + "/" + rel
}

// locate maps one non-recursive pattern onto (dir, import path).
func (r *Resolver) locate(pat string) (dir, pkgPath string, ok bool) {
	if pat == "." || strings.HasPrefix(pat, "./") || strings.HasPrefix(pat, "../") || r.vfs.IsAbs(pat) {
		dir = r.vfs.Join(r.dir, strings.TrimPrefix(pat, "./"))
		if r.vfs.IsAbs(pat) {
			dir = pat
		}
		if !r.vfs.IsDir(dir) {
			return "", "", false
		}
		return dir, r.pkgPathFor(dir), true
	}
	// A bare pattern is an import path, as it is for the go command — never a directory. Falling back
	// to a directory read looks generous, but it means the same pattern names different packages under
	// the two strategies: `go list internal/foo` reports "not in std", while a directory read quietly
	// succeeds with a different identity.
	if d, p, found := r.ResolveImport(pat); found {
		return d, p, true
	}

	return "", "", false
}

// pkgPathFor derives an import path for a directory, from the module that actually contains it.
//
// Deriving it from the main module instead is wrong the moment a directory belongs to a nested or
// workspace module: the path comes out well-formed but names a package that does not exist. Since
// codescan keys definitions, provenance and x-go-package on the import path, a plausible-looking
// wrong one is worse than an obviously wrong one.
func (r *Resolver) pkgPathFor(dir string) string {
	mod := r.nearestModule(dir)
	if mod.root == "" || mod.path == "" {
		return r.pathlessID(dir)
	}

	rel, ok := r.vfs.HasSubdir(mod.root, dir)
	if !ok {
		return dir
	}

	return joinPkgPath(mod.path, rel)
}

// pathlessID names a package in a tree that has no module.
//
// The go command simply refuses such a tree, so there is nothing to be faithful to; what matters is
// that the answer is not an absolute path. A package path ends up in x-go-package and in the
// collision-naming machinery, and "/home/someone/scratch/api" in a published spec describes the
// machine that produced it rather than the code it documents.
func (r *Resolver) pathlessID(dir string) string {
	rel, ok := r.vfs.HasSubdir(r.dir, dir)
	if !ok || rel == "." || rel == "" {
		return path.Base(dir)
	}

	return rel
}

// nearestModule finds the module a directory belongs to, walking up to the first go.mod.
func (r *Resolver) nearestModule(dir string) moduleAt {
	if m, ok := r.nearestMod[dir]; ok {
		return m
	}

	d, found := dir, moduleAt{}
	for range maxParentWalk {
		// Stop at the boundary, whether or not the go.mod there names a module: a directory under an
		// unreadable go.mod belongs to that module, not to the one further up, so borrowing the enclosing
		// module's path would invent an import path rather than admit there is none.
		if r.isModuleRoot(d) {
			found = moduleAt{root: d, path: r.modulePathAt(d)}

			break
		}
		// The main module is the backstop: its root may sit above the bounded walk, and stopping there
		// keeps a directory inside it from being reported as module-less.
		if d == r.modRoot {
			found = moduleAt{root: r.modRoot, path: r.modPath}

			break
		}

		parent := r.vfs.Join(d, "..")
		if parent == d || d == "." || d == "/" {
			break
		}
		d = parent
	}

	r.nearestMod[dir] = found

	return found
}

// isModuleRoot reports whether dir is a module boundary.
//
// The test is the existence of a go.mod file, not whether it parses: cmd/go stops a walk on the stat
// alone (modload/search.go, "Stop at module boundaries"). An empty or broken go.mod still marks a
// different module, and treating it as ordinary meant walking into it and labelling its packages with
// the enclosing module's path.
func (r *Resolver) isModuleRoot(dir string) bool {
	_, err := r.vfs.ReadFile(r.vfs.Join(dir, "go.mod"))

	return err == nil
}

func (r *Resolver) hasGoFiles(dir string) bool {
	infos, err := r.vfs.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, i := range infos {
		if !i.IsDir() && strings.HasSuffix(i.Name(), ".go") {
			return true
		}
	}
	return false
}

func joinPkgPath(base, rel string) string {
	if rel == "" || rel == "." {
		return base
	}
	if base == "" {
		return rel
	}
	return path.Join(base, rel)
}

// IsStdlibPath reports whether an import path names a standard-library package: its first segment
// carries no dot, so it can never be a module path.
func IsStdlibPath(importPath string) bool {
	return !strings.Contains(firstSegment(importPath), ".")
}

func firstSegment(p string) string {
	before, _, _ := strings.Cut(p, "/")

	return before
}
