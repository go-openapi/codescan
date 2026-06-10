// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"fmt"
	"go/types"

	"github.com/go-openapi/codescan/internal/builders/common"
	"github.com/go-openapi/codescan/internal/builders/resolvers"
	"github.com/go-openapi/codescan/internal/builders/schema"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/logger"
	"github.com/go-openapi/codescan/internal/scanner"
	oaispec "github.com/go-openapi/spec"
)

const inBody = "body"

// Builder constructs OAS v2 parameter entries for one
// `swagger:parameters` declaration and writes them onto the matching
// operations. Embeds *common.Builder for shared state (Ctx, Decl,
// PostDeclarations, diagnostics, ParseBlocks cache).
type Builder struct {
	*common.Builder
}

// NewBuilder constructs an initialized [Builder] bound to
// ctx and decl. The embedded common.Builder owns the diagnostic
// sink, the post-declaration list, and the per-comment-group parse
// cache.
func NewBuilder(ctx *scanner.ScanCtx, decl *scanner.EntityDecl) *Builder {
	return &Builder{
		Builder: common.New(ctx, decl),
	}
}

func (p *Builder) Build(operations map[string]*oaispec.Operation) error {
	// check if there is a swagger:parameters tag that is followed by one or more words,
	// these words are the ids of the operations this parameter struct applies to
	// once type name is found convert it to a schema, by looking up the schema in the
	// parameters dictionary that got passed into this parse method
	for _, opid := range p.Decl.OperationIDs() {
		operation, ok := operations[opid]
		if !ok {
			operation = new(oaispec.Operation)
			operations[opid] = operation
			operation.ID = opid
		}
		logger.DebugLogf(p.Ctx.Debug(), "building parameters for: %s", opid)

		// analyze struct body for fields etc
		// each exported struct field:
		// * gets a type mapped to a go primitive
		// * perhaps gets a format
		// * has to document the validations that apply for the type and the field
		// * when the struct field points to a model it becomes a ref: #/definitions/ModelName
		// * comments that aren't tags is used as the description
		if err := p.buildFromType(p.Decl.ObjType(), operation, make(map[string]oaispec.Parameter)); err != nil {
			return err
		}
	}

	return nil
}

func (p *Builder) buildFromType(otpe types.Type, op *oaispec.Operation, seen map[string]oaispec.Parameter) error {
	switch tpe := otpe.(type) {
	case *types.Pointer:
		return p.buildFromType(tpe.Elem(), op, seen)
	case *types.Named:
		return p.buildNamedType(tpe, op, seen)
	case *types.Alias:
		logger.DebugLogf(p.Ctx.Debug(), "alias(parameters.buildFromType): got alias %v to %v", tpe, tpe.Rhs())
		return p.buildAlias(tpe, op, seen)
	default:
		return fmt.Errorf("unhandled type (%T): %s: %w", otpe, tpe.String(), ErrParameters)
	}
}

func (p *Builder) buildNamedType(tpe *types.Named, op *oaispec.Operation, seen map[string]oaispec.Parameter) error {
	o := tpe.Obj()
	if resolvers.IsAny(o) || resolvers.IsStdError(o) {
		return fmt.Errorf("%s type not supported in the context of a parameters section definition: %w", o.Name(), ErrParameters)
	}
	resolvers.MustNotBeABuiltinType(o)

	switch stpe := o.Type().Underlying().(type) {
	case *types.Struct:
		logger.DebugLogf(p.Ctx.Debug(), "build from named type %s: %T", o.Name(), tpe)
		if decl, found := p.Ctx.DeclForType(o.Type()); found {
			return p.buildFromStruct(decl, stpe, op, seen)
		}

		return p.buildFromStruct(p.Decl, stpe, op, seen)
	default:
		return fmt.Errorf("unhandled type (%T): %s: %w", stpe, o.Type().Underlying().String(), ErrParameters)
	}
}

func (p *Builder) buildAlias(tpe *types.Alias, op *oaispec.Operation, seen map[string]oaispec.Parameter) error {
	o := tpe.Obj()
	if resolvers.IsAny(o) || resolvers.IsStdError(o) {
		return fmt.Errorf("%s type not supported in the context of a parameters section definition: %w", o.Name(), ErrParameters)
	}
	resolvers.MustNotBeABuiltinType(o)
	resolvers.MustHaveRightHandSide(tpe)

	// `swagger:parameters` declares a parameter SET, not a model. Neither
	// the alias decl nor any chain link of its target surfaces as a
	// `definitions` entry — the fields of the unaliased target become the
	// operation's parameters. There is no mode-specific behaviour for this
	// case: TransparentAliases takes the same path as Default and
	// RefAliases. The mode flags only affect alias *use* sites (field /
	// element), not the top-level parameter-set declaration.
	//
	// Recursion handles alias chains naturally: buildFromType dispatches
	// back here for any chain link whose RHS is itself an alias.
	return p.buildFromType(tpe.Rhs(), op, seen)
}

func (p *Builder) buildFromField(fld *types.Var, tpe types.Type, typable ifaces.SwaggerTypable, seen map[string]oaispec.Parameter) error {
	logger.DebugLogf(p.Ctx.Debug(), "build from field %s: %T", fld.Name(), tpe)

	switch ftpe := tpe.(type) {
	case *types.Basic:
		return resolvers.SwaggerSchemaForType(ftpe.Name(), typable)
	case *types.Struct:
		return p.buildFromFieldStruct(ftpe, typable)
	case *types.Pointer:
		return p.buildFromField(fld, ftpe.Elem(), typable, seen)
	case *types.Interface:
		return p.buildFromFieldInterface(ftpe, typable)
	case *types.Array:
		return p.buildFromField(fld, ftpe.Elem(), typable.Items(), seen)
	case *types.Slice:
		return p.buildFromField(fld, ftpe.Elem(), typable.Items(), seen)
	case *types.Map:
		return p.buildFromFieldMap(ftpe, typable)
	case *types.Named:
		return p.buildNamedField(ftpe, typable)
	case *types.Alias:
		logger.DebugLogf(p.Ctx.Debug(), "alias(parameters.buildFromField): got alias %v to %v", ftpe, ftpe.Rhs())
		return p.buildFieldAlias(ftpe, typable, fld, seen)
	default:
		return fmt.Errorf("unknown type for %s: %T: %w", fld.String(), fld.Type(), ErrParameters)
	}
}

func (p *Builder) buildFromFieldStruct(tpe *types.Struct, typable ifaces.SwaggerTypable) error {
	sb := schema.NewBuilder(p.Ctx, p.Decl)
	if err := sb.Build(schema.OptionFor(tpe, typable)); err != nil {
		return err
	}
	for _, d := range sb.PostDeclarations() {
		p.AppendPostDecl(d)
	}

	return nil
}

func (p *Builder) buildFromFieldMap(ftpe *types.Map, typable ifaces.SwaggerTypable) error {
	// Map fields are only legal under in=body — paramTypable.Schema()
	// returns nil for non-body. No SimpleSchema variant needed.
	sch := new(oaispec.Schema)
	typable.Schema().Typed("object", "").AdditionalProperties = &oaispec.SchemaOrBool{
		Schema: sch,
	}

	sb := schema.NewBuilder(p.Ctx, p.Decl)
	if err := sb.Build(schema.WithType(
		ftpe.Elem(),
		schema.NewTypable(sch, typable.Level()+1, p.Ctx.SkipExtensions())),
	); err != nil {
		return err
	}

	// Propagate the sub-builder's PostDeclarations so a model
	// discovered only through the map's value type (no
	// swagger:model annotation, no other reference site) makes it
	// into the spec's definitions section. Every sibling
	// buildFromFieldXxx method does the same; this loop went
	// missing in M2.5's schema-builder factor-out — see the
	// parameters-map-postdecl fixture.
	for _, d := range sb.PostDeclarations() {
		p.AppendPostDecl(d)
	}

	return nil
}

func (p *Builder) buildFromFieldInterface(tpe *types.Interface, typable ifaces.SwaggerTypable) error {
	sb := schema.NewBuilder(p.Ctx, p.Decl)
	if err := sb.Build(schema.OptionFor(tpe, typable)); err != nil {
		return err
	}

	for _, d := range sb.PostDeclarations() {
		p.AppendPostDecl(d)
	}

	return nil
}

func (p *Builder) buildNamedField(ftpe *types.Named, typable ifaces.SwaggerTypable) error {
	o := ftpe.Obj()
	if resolvers.IsAny(o) {
		// e.g. Field interface{} or Field any
		return nil
	}
	if resolvers.IsStdError(o) {
		return fmt.Errorf("%s type not supported in the context of a parameter definition: %w", o.Name(), ErrParameters)
	}
	resolvers.MustNotBeABuiltinType(o)

	decl, found := p.Ctx.DeclForType(o.Type())
	if !found {
		return fmt.Errorf("unable to find package and source file for: %s: %w", ftpe.String(), ErrParameters)
	}

	if resolvers.IsStdTime(o) {
		typable.Typed("string", "date-time")
		return nil
	}

	if sfnm, isf := strfmtFromDoc(p.ParseBlocks(decl.Comments)); isf {
		typable.Typed("string", sfnm)
		return nil
	}

	sb := schema.NewBuilder(p.Ctx, decl)
	sb.InferNames()
	if err := sb.Build(schema.OptionFor(decl.ObjType(), typable)); err != nil {
		return err
	}

	for _, d := range sb.PostDeclarations() {
		p.AppendPostDecl(d)
	}

	return nil
}

func (p *Builder) buildFieldAlias(tpe *types.Alias, typable ifaces.SwaggerTypable, fld *types.Var, seen map[string]oaispec.Parameter) error {
	o := tpe.Obj()
	if resolvers.IsAny(o) {
		// e.g. Field interface{} or Field any
		_ = typable.Schema()

		return nil // just leave an empty schema
	}
	if resolvers.IsStdError(o) {
		return fmt.Errorf("%s type not supported in the context of a parameter definition: %w", o.Name(), ErrParameters)
	}
	resolvers.MustNotBeABuiltinType(o)
	resolvers.MustHaveRightHandSide(tpe)

	// TransparentAliases supersedes annotation at use sites — dissolve
	// to the unaliased target via the schema sub-builder.
	if p.Ctx.TransparentAliases() {
		sb := schema.NewBuilder(p.Ctx, p.Decl)
		if err := sb.Build(schema.OptionFor(tpe.Rhs(), typable)); err != nil {
			return err
		}
		for _, d := range sb.PostDeclarations() {
			p.AppendPostDecl(d)
		}
		return nil
	}

	decl, ok := p.Ctx.GetModel(o.Pkg().Path(), o.Name())
	if !ok {
		return fmt.Errorf("can't find source file for aliased type: %v -> %v: %w", tpe, tpe.Rhs(), ErrParameters)
	}

	// Non-body parameters are SimpleSchema targets and cannot carry $ref —
	// always expand the alias to its unaliased target regardless of
	// annotation. Walking through every alias layer (types.Unalias)
	// dissolves chains fully in one step.
	if typable.In() != inBody {
		return p.buildFromField(fld, types.Unalias(tpe), typable, seen)
	}

	// Body field: annotation gates first-class identity at the use site
	// (parameters analogue of the schema-builder R6 rule).
	//
	//   - annotated   alias → $ref preserves the alias name; the alias
	//     gets its own definition via MakeRef's AppendPostDecl side effect.
	//   - unannotated alias → dissolve to the unaliased target (full
	//     chain collapse via types.Unalias); the alias produces no
	//     definition entry.
	//
	// The mode flag (RefAliases vs Default) only affects the shape of the
	// alias decl's OWN definition downstream — it does not change the
	// field-site $ref target, which is gated entirely by annotation.
	if decl.HasModelAnnotation() {
		return p.MakeRef(decl, typable)
	}

	return p.buildFromField(fld, types.Unalias(tpe), typable, seen)
}

func (p *Builder) buildFromStruct(decl *scanner.EntityDecl, tpe *types.Struct, op *oaispec.Operation, seen map[string]oaispec.Parameter) error {
	numFields := tpe.NumFields()

	if numFields == 0 {
		return nil
	}

	sequence := make([]string, 0, numFields)
	for fld := range tpe.Fields() {
		if fld.Embedded() {
			if err := p.buildFromType(fld.Type(), op, seen); err != nil {
				return err
			}
			continue
		}

		name, err := p.processParamField(fld, decl, seen)
		if err != nil {
			return err
		}

		if name != "" {
			sequence = append(sequence, name)
		}
	}

	for _, k := range sequence {
		p := seen[k]
		for i, v := range op.Parameters {
			if v.Name == k {
				op.Parameters = append(op.Parameters[:i], op.Parameters[i+1:]...)
				break
			}
		}
		op.Parameters = append(op.Parameters, p)
	}

	return nil
}

// processParamField processes a single non-embedded struct field for parameter building.
// Returns the parameter name if the field was processed, or "" if it was skipped.
func (p *Builder) processParamField(fld *types.Var, decl *scanner.EntityDecl, seen map[string]oaispec.Parameter) (string, error) {
	if !fld.Exported() {
		logger.DebugLogf(p.Ctx.Debug(), "skipping field %s because it's not exported", fld.Name())
		return "", nil
	}

	afld := resolvers.FindASTField(decl.File, fld.Pos())
	if afld == nil {
		logger.DebugLogf(p.Ctx.Debug(), "can't find source associated with %s", fld.String())
		return "", nil
	}

	signals := scanFieldDocSignals(p.ParseBlocks(afld.Doc), afld.Doc)

	if signals.ignored {
		return "", nil
	}

	name, ignore, _, _, err := resolvers.ParseJSONTag(afld)
	if err != nil {
		return "", err
	}
	if ignore {
		return "", nil
	}

	in := "query"
	if signals.inSet {
		in = signals.in
	}

	ps := seen[name]
	ps.In = in
	var pty ifaces.SwaggerTypable = paramTypable{&ps, p.Ctx.SkipExtensions()}
	if in == inBody {
		pty = schema.NewTypable(pty.Schema(), 0, p.Ctx.SkipExtensions())
	}

	if in == "formData" && signals.file {
		pty.Typed("file", "")
	} else if err := p.buildFromField(fld, fld.Type(), pty, seen); err != nil {
		return "", err
	}

	if signals.strfmtSet {
		ps.Typed("string", signals.strfmt)
		ps.Ref = oaispec.Ref{}
		ps.Items = nil
	}

	if err := p.applyBlockToField(afld, &ps); err != nil {
		return "", err
	}
	if ps.In == "path" {
		ps.Required = true
	}

	if ps.Name == "" {
		ps.Name = name
	}

	if name != fld.Name() {
		resolvers.AddExtension(&ps.VendorExtensible, "x-go-name", fld.Name(), p.Ctx.SkipExtensions())
	}

	seen[name] = ps
	return name, nil
}
