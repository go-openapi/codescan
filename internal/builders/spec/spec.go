// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"iter"
	"sort"
	"strconv"

	"github.com/go-openapi/codescan/internal/builders/operations"
	"github.com/go-openapi/codescan/internal/builders/parameters"
	"github.com/go-openapi/codescan/internal/builders/responses"
	"github.com/go-openapi/codescan/internal/builders/routes"
	"github.com/go-openapi/codescan/internal/builders/schema"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	oaispec "github.com/go-openapi/spec"
)

// ErrInternalPanic is the base error for a builder panic recovered while
// processing a single declaration. Rather than surfacing a raw Go stack
// trace, the scan names the offending source declaration (file:line) and
// aborts with this error. See go-swagger/go-swagger#2886.
var ErrInternalPanic = errors.New("internal panic during build")

type Builder struct {
	scanModels  bool
	input       *oaispec.Swagger
	ctx         *scanner.ScanCtx
	discovered  []*scanner.EntityDecl
	definitions map[string]oaispec.Schema
	responses   map[string]oaispec.Response
	operations  map[string]*oaispec.Operation
}

func NewBuilder(input *oaispec.Swagger, sc *scanner.ScanCtx, scanModels bool) *Builder {
	if input == nil {
		input = new(oaispec.Swagger)
		input.Swagger = "2.0"
	}

	if input.Paths == nil {
		input.Paths = new(oaispec.Paths)
	}
	if input.Definitions == nil {
		input.Definitions = make(map[string]oaispec.Schema)
	}
	if input.Responses == nil {
		input.Responses = make(map[string]oaispec.Response)
	}
	if input.Extensions == nil {
		input.Extensions = make(oaispec.Extensions)
	}

	return &Builder{
		ctx:         sc,
		input:       input,
		scanModels:  scanModels,
		operations:  collectOperationsFromInput(input),
		definitions: input.Definitions,
		responses:   input.Responses,
	}
}

func (s *Builder) Build() (*oaispec.Swagger, error) {
	// Resolve same-package duplicate swagger:model names up front so that
	// every later DefKey() / MakeRef() observes a consistent, conflict-free
	// key for each declaration (D-4). Must run before anything emits a ref.
	s.resolveSamePackageDuplicates()

	// this initial scan step is skipped if !scanModels.
	// Discovered dependencies should however be resolved.
	if err := s.buildModels(); err != nil {
		return nil, err
	}

	if err := s.buildParameters(); err != nil {
		return nil, err
	}

	if err := s.buildResponses(); err != nil {
		return nil, err
	}

	// build definitions dictionary
	if err := s.buildDiscovered(); err != nil {
		return nil, err
	}

	if err := s.buildRoutes(); err != nil {
		return nil, err
	}

	if err := s.buildOperations(); err != nil {
		return nil, err
	}

	if err := s.buildMeta(); err != nil {
		return nil, err
	}

	// Cross-ref linkage: parameter anchors are resolved here, once paths,
	// methods and final array indices are all known (see emitParameterAnchors).
	// Param anchors live under /paths/.../parameters/{i}, independent of the
	// definition-name reduction below.
	s.emitParameterAnchors()

	// Final stage: shorten the fully-qualified, collision-proof
	// definition keys produced during discovery back to user-facing
	// names and re-point every $ref. Runs last because buildRoutes /
	// buildOperations also emit definition refs. See
	// .claude/plans/name-identity-cyclic-ref.md §9/§12.
	s.reduceDefinitionNames()

	if s.input.Swagger == "" {
		s.input.Swagger = "2.0"
	}

	return s.input, nil
}

// emitParameterAnchors fires the deferred /paths/{path}/{method}/parameters/{i}
// anchors. Parameters are built before their operation is bound to a path and
// before the parameter array order is final, so the parameters builder only
// captured (opid → name → position) via ScanCtx.RecordParamOrigin; here the
// absolute pointer is assembled from the finished paths tree. Finer parameter
// sub-nodes resolve to the parameter (or the operation) anchor.
func (s *Builder) emitParameterAnchors() {
	if !s.ctx.OriginEnabled() || s.input.Paths == nil {
		return
	}

	for path, item := range s.input.Paths.Paths {
		for method, op := range operationsByMethod(&item) {
			for i, prm := range op.Parameters {
				pos, ok := s.ctx.ParamOrigin(op.ID, prm.Name)
				if !ok {
					continue
				}
				s.ctx.RecordOrigin(
					scanner.JSONPointer("paths", path, method, "parameters", strconv.Itoa(i)),
					pos,
				)
			}
		}
	}
}

// operationsByMethod yields each populated (lowercase method → operation) slot
// of a path item, in a stable order. (Distinct from reduce.go's operationsOf,
// which returns the bare operation pointers for ref rewriting; here the method
// name is needed to build the /paths/{path}/{method}/... anchor pointer.)
func operationsByMethod(item *oaispec.PathItem) iter.Seq2[string, *oaispec.Operation] {
	return func(yield func(string, *oaispec.Operation) bool) {
		slots := []struct {
			method string
			op     *oaispec.Operation
		}{
			{"get", item.Get},
			{"put", item.Put},
			{"post", item.Post},
			{"delete", item.Delete},
			{"options", item.Options},
			{"head", item.Head},
			{"patch", item.Patch},
		}
		for _, slot := range slots {
			if slot.op == nil {
				continue
			}
			if !yield(slot.method, slot.op) {
				return
			}
		}
	}
}

// guard runs a per-declaration build step under panic recovery. On panic it
// emits a located scan.internal-panic diagnostic naming the offending source
// (pos + what), then returns an aborting error wrapping ErrInternalPanic — so
// a builder bug surfaces as a clean, located failure instead of a raw stack
// trace (#2886). A non-panicking run is transparent. pos/what identify the
// declaration being built; the spec.Builder build loops wrap each per-decl
// step. Panics outside any decl loop are caught by the Run backstop.
func (s *Builder) guard(pos token.Position, what string, run func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if cb := s.ctx.OnDiagnostic(); cb != nil {
				cb(grammar.Errorf(pos, grammar.CodeInternalPanic,
					"panic while building %s: %v", what, r))
			}
			err = fmt.Errorf("%w: %s: %s: %v", ErrInternalPanic, pos, what, r)
		}
	}()

	return run()
}

// declLabel names an EntityDecl for a diagnostic — the fully-qualified Go
// type when the package path is known, else the bare identifier.
func declLabel(d *scanner.EntityDecl) string {
	if d.Pkg != nil && d.Pkg.PkgPath != "" {
		return d.Pkg.PkgPath + "." + d.Ident.Name
	}

	return d.Ident.Name
}

func (s *Builder) buildDiscovered() error {
	// loop over discovered until all the items are in definitions
	keepGoing := len(s.discovered) > 0
	for keepGoing {
		var queue []*scanner.EntityDecl
		// Dedupe by name within this pass. The same decl can appear
		// multiple times in s.discovered (one entry per reference
		// site that called AppendPostDecl); without this, both copies
		// get queued and Build runs twice, each appending to the
		// existing schema's AllOf and producing doubled entries.
		queued := make(map[string]struct{})
		for _, d := range s.discovered {
			// Dedup by the fully-qualified identity key (pkgpath/name),
			// matching the definitions-map key written by the schema
			// builder. A cycle / re-discovery of the SAME decl resolves
			// to the same key and is skipped (reuse); two distinct decls
			// that merely share a short name now get distinct keys and
			// both build. See name-identity design §9.1/§12.1.
			key := d.DefKey()
			if _, alreadyDone := s.definitions[key]; alreadyDone {
				continue
			}
			if _, dupInPass := queued[key]; dupInPass {
				continue
			}
			queued[key] = struct{}{}
			queue = append(queue, d)
		}
		s.discovered = nil
		for _, sd := range queue {
			if err := s.guard(s.ctx.PosOf(sd.Ident.Pos()), declLabel(sd), func() error {
				return s.buildDiscoveredSchema(sd)
			}); err != nil {
				return err
			}
		}
		keepGoing = len(s.discovered) > 0
	}

	return nil
}

func (s *Builder) buildDiscoveredSchema(decl *scanner.EntityDecl) error {
	sb := schema.NewBuilder(s.ctx, decl)
	sb.SetDiscovered(s.discovered)

	// Cross-ref linkage: initiate the base pointer for this definition so the
	// schema builder path-joins its members (properties, …) under it, and anchor
	// the definition node itself to its type declaration.
	var defPtr string
	opts := []schema.Option{schema.WithDefinitions(s.definitions)}
	if s.ctx.OriginEnabled() {
		if name, _ := decl.Names(); name != "" {
			defPtr = scanner.JSONPointer("definitions", name)
			opts = append(opts, schema.WithPath(defPtr))
		}
	}

	if err := sb.Build(opts...); err != nil {
		return err
	}

	if defPtr != "" {
		s.ctx.RecordOrigin(defPtr, s.ctx.PosOf(decl.Ident.Pos()))
	}

	s.discovered = append(s.discovered, sb.PostDeclarations()...)

	return nil
}

func (s *Builder) buildMeta() error {
	parser := grammar.NewParser(s.ctx.FileSet(),
		grammar.WithSingleLineCommentAsDescription(s.ctx.SingleLineCommentAsDescription()))
	for cg := range s.ctx.Meta() {
		block := parser.Parse(cg)
		if err := applyMetaBlock(s.input, block); err != nil {
			return err
		}

		// Cross-ref linkage: anchor the info node to its swagger:meta block,
		// then each meta keyword to its own line (the top-level fields —
		// host/basePath/consumes/… — have no ancestor anchor otherwise, since
		// /info is their sibling, not parent).
		if s.ctx.OriginEnabled() {
			s.ctx.RecordOrigin(scanner.JSONPointer("info"), s.ctx.PosOf(cg.Pos()))
			s.recordMetaOrigins(block)
		}
	}

	return nil
}

// recordMetaOrigins anchors each meta keyword in block to its source line. The
// keyword→pointer knowledge lives in the grammar ([grammar.PointerPath]); meta
// pointers are absolute (Info.* under /info, the rest at the document root), so
// there is no base to prepend.
func (s *Builder) recordMetaOrigins(block grammar.Block) {
	for p := range block.Properties() {
		if p.ItemsDepth != 0 {
			continue
		}
		if segs, ok := grammar.PointerPath(p.Keyword, grammar.CtxMeta); ok {
			s.ctx.RecordOrigin(scanner.JSONPointer(segs...), p.Pos)
		}
	}
}

func (s *Builder) buildOperations() error {
	for pp := range s.ctx.Operations() {
		if err := s.guard(s.ctx.PosOf(pp.Pos), "operation "+pp.Method+" "+pp.Path, func() error {
			ob := operations.NewBuilder(s.ctx, pp, s.operations)

			return ob.Build(s.input.Paths)
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Builder) buildRoutes() error {
	// build paths dictionary
	for pp := range s.ctx.Routes() {
		if err := s.guard(s.ctx.PosOf(pp.Pos), "route "+pp.Method+" "+pp.Path, func() error {
			rb := routes.NewBuilder(
				s.ctx,
				pp,
				routes.Inputs{
					Responses:   s.responses,
					Operations:  s.operations,
					Definitions: s.definitions,
				},
			)

			return rb.Build(s.input.Paths)
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Builder) buildResponses() error {
	// build responses dictionary
	for decl := range s.ctx.Responses() {
		if err := s.guard(s.ctx.PosOf(decl.Ident.Pos()), declLabel(decl), func() error {
			rb := responses.NewBuilder(s.ctx, decl)
			if err := rb.Build(s.responses); err != nil {
				return err
			}
			s.discovered = append(s.discovered, rb.PostDeclarations()...)

			// Cross-ref linkage: anchor the top-level response node to its
			// swagger:response declaration; headers/body resolve to it.
			if s.ctx.OriginEnabled() {
				if name, _ := decl.ResponseNames(); name != "" {
					s.ctx.RecordOrigin(scanner.JSONPointer("responses", name), s.ctx.PosOf(decl.Ident.Pos()))
				}
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Builder) buildParameters() error {
	// build parameters dictionary
	for decl := range s.ctx.Parameters() {
		if err := s.guard(s.ctx.PosOf(decl.Ident.Pos()), declLabel(decl), func() error {
			pb := parameters.NewBuilder(s.ctx, decl)
			if err := pb.Build(s.operations); err != nil {
				return err
			}
			s.discovered = append(s.discovered, pb.PostDeclarations()...)

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// resolveSamePackageDuplicates detects two distinct annotated models in
// the SAME package that claim the same definition name — necessarily via
// a `swagger:model <name>` override, since Go type names are unique per
// package. The first (in a deterministic order) keeps the name; later
// ones have their override suppressed (reverting to the Go type name) and
// get a diagnostic. This is the build-side half of D-4; cross-package
// same-name collisions are handled later by the reduce stage. See
// .claude/plans/name-identity-cyclic-ref.md §9.1/§12.1.
func (s *Builder) resolveSamePackageDuplicates() {
	models := make([]*scanner.EntityDecl, 0)
	for _, d := range s.ctx.Models() {
		models = append(models, d)
	}
	// Deterministic order so "first wins" is stable across runs: by key,
	// then by Go name within a colliding group.
	sort.Slice(models, func(i, j int) bool {
		ki, kj := models[i].DefKey(), models[j].DefKey()
		if ki != kj {
			return ki < kj
		}
		return models[i].Ident.Name < models[j].Ident.Name
	})

	seen := make(map[string]*scanner.EntityDecl, len(models))
	for _, d := range models {
		key := d.DefKey()
		if first, dup := seen[key]; dup && first.Ident != d.Ident {
			d.SuppressModelOverride()
			if onDiag := s.ctx.OnDiagnostic(); onDiag != nil {
				_, goName := d.Names()
				onDiag(grammar.Warnf(
					s.ctx.PosOf(d.Spec.Pos()),
					grammar.CodeDuplicateModelName,
					"duplicate swagger:model name %q in package %q (already used by type %q); using Go name %q instead",
					leafName(key), d.Obj().Pkg().Path(), first.Ident.Name, goName,
				))
			}
			continue
		}
		seen[key] = d
	}
}

func (s *Builder) buildModels() error {
	// build models dictionary
	if !s.scanModels {
		return nil
	}

	for _, decl := range s.ctx.Models() {
		if err := s.guard(s.ctx.PosOf(decl.Ident.Pos()), declLabel(decl), func() error {
			return s.buildDiscoveredSchema(decl)
		}); err != nil {
			return err
		}
	}

	return s.joinExtraModels()
}

func (s *Builder) joinExtraModels() error {
	l := s.ctx.NumExtraModels()
	if l == 0 {
		return nil
	}

	tmp := make(map[*ast.Ident]*scanner.EntityDecl, l)
	for k, v := range s.ctx.ExtraModels() {
		tmp[k] = v
		s.ctx.MoveExtraToModel(k)
	}

	// process extra models and see if there is any reference to a new extra one
	for _, decl := range tmp {
		if err := s.buildDiscoveredSchema(decl); err != nil {
			return err
		}
	}

	if s.ctx.NumExtraModels() > 0 {
		return s.joinExtraModels()
	}

	return nil
}

func collectOperationsFromInput(input *oaispec.Swagger) map[string]*oaispec.Operation {
	operations := make(map[string]*oaispec.Operation)
	if input == nil || input.Paths == nil {
		return operations
	}

	for _, pth := range input.Paths.Paths {
		if pth.Get != nil {
			operations[pth.Get.ID] = pth.Get
		}
		if pth.Post != nil {
			operations[pth.Post.ID] = pth.Post
		}
		if pth.Put != nil {
			operations[pth.Put.ID] = pth.Put
		}
		if pth.Patch != nil {
			operations[pth.Patch.ID] = pth.Patch
		}
		if pth.Delete != nil {
			operations[pth.Delete.ID] = pth.Delete
		}
		if pth.Head != nil {
			operations[pth.Head.ID] = pth.Head
		}
		if pth.Options != nil {
			operations[pth.Options.ID] = pth.Options
		}
	}

	return operations
}
