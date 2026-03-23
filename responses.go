// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/go-openapi/spec"
)

type responseTypable struct {
	in       string
	header   *spec.Header
	response *spec.Response
	skipExt  bool
}

func (ht responseTypable) In() string { return ht.in }

func (ht responseTypable) Level() int { return 0 }

func (ht responseTypable) Typed(tpe, format string) {
	ht.header.Typed(tpe, format)
}

func bodyTypable(in string, schema *spec.Schema, skipExt bool) (swaggerTypable, *spec.Schema) { //nolint:ireturn // polymorphic by design
	if in == bodyTag {
		// get the schema for items on the schema property
		if schema == nil {
			schema = new(spec.Schema)
		}
		if schema.Items == nil {
			schema.Items = new(spec.SchemaOrArray)
		}
		if schema.Items.Schema == nil {
			schema.Items.Schema = new(spec.Schema)
		}
		schema.Typed("array", "")
		return schemaTypable{schema.Items.Schema, 1, skipExt}, schema
	}
	return nil, nil
}

func (ht responseTypable) Items() swaggerTypable { //nolint:ireturn // polymorphic by design
	bdt, schema := bodyTypable(ht.in, ht.response.Schema, ht.skipExt)
	if bdt != nil {
		ht.response.Schema = schema
		return bdt
	}

	if ht.header.Items == nil {
		ht.header.Items = new(spec.Items)
	}
	ht.header.Type = "array"
	return itemsTypable{ht.header.Items, 1, "header"}
}

func (ht responseTypable) SetRef(ref spec.Ref) {
	// having trouble seeing the usefulness of this one here
	ht.Schema().Ref = ref
}

func (ht responseTypable) Schema() *spec.Schema {
	if ht.response.Schema == nil {
		ht.response.Schema = new(spec.Schema)
	}
	return ht.response.Schema
}

func (ht responseTypable) SetSchema(schema *spec.Schema) {
	ht.response.Schema = schema
}

func (ht responseTypable) CollectionOf(items *spec.Items, format string) {
	ht.header.CollectionOf(items, format)
}

func (ht responseTypable) AddExtension(key string, value any) {
	ht.response.AddExtension(key, value)
}

func (ht responseTypable) WithEnum(values ...any) {
	ht.header.WithEnum(values)
}

func (ht responseTypable) WithEnumDescription(_ string) {
	// no
}

type headerValidations struct {
	current *spec.Header
}

func (sv headerValidations) SetMaximum(val float64, exclusive bool) {
	sv.current.Maximum = &val
	sv.current.ExclusiveMaximum = exclusive
}

func (sv headerValidations) SetMinimum(val float64, exclusive bool) {
	sv.current.Minimum = &val
	sv.current.ExclusiveMinimum = exclusive
}

func (sv headerValidations) SetMultipleOf(val float64) {
	sv.current.MultipleOf = &val
}

func (sv headerValidations) SetMinItems(val int64) {
	sv.current.MinItems = &val
}

func (sv headerValidations) SetMaxItems(val int64) {
	sv.current.MaxItems = &val
}

func (sv headerValidations) SetMinLength(val int64) {
	sv.current.MinLength = &val
}

func (sv headerValidations) SetMaxLength(val int64) {
	sv.current.MaxLength = &val
}

func (sv headerValidations) SetPattern(val string) {
	sv.current.Pattern = val
}

func (sv headerValidations) SetUnique(val bool) {
	sv.current.UniqueItems = val
}

func (sv headerValidations) SetCollectionFormat(val string) {
	sv.current.CollectionFormat = val
}

func (sv headerValidations) SetEnum(val string) {
	sv.current.Enum = parseEnum(val, &spec.SimpleSchema{Type: sv.current.Type, Format: sv.current.Format})
}

func (sv headerValidations) SetDefault(val any) { sv.current.Default = val }

func (sv headerValidations) SetExample(val any) { sv.current.Example = val }

type responseBuilder struct {
	ctx       *scanCtx
	decl      *entityDecl
	postDecls []*entityDecl
}

func (r *responseBuilder) Build(responses map[string]spec.Response) error {
	// check if there is a swagger:response tag that is followed by one or more words,
	// these words are the ids of the operations this parameter struct applies to
	// once type name is found convert it to a schema, by looking up the schema in the
	// parameters dictionary that got passed into this parse method

	name, _ := r.decl.ResponseNames()
	response := responses[name]
	debugLogf(r.ctx.debug, "building response: %s", name)

	// analyze doc comment for the model
	sp := new(sectionedParser)
	sp.setDescription = func(lines []string) { response.Description = joinDropLast(lines) }
	if err := sp.Parse(r.decl.Comments); err != nil {
		return err
	}

	// analyze struct body for fields etc
	// each exported struct field:
	// * gets a type mapped to a go primitive
	// * perhaps gets a format
	// * has to document the validations that apply for the type and the field
	// * when the struct field points to a model it becomes a ref: #/definitions/ModelName
	// * comments that aren't tags is used as the description
	if err := r.buildFromType(r.decl.ObjType(), &response, make(map[string]bool)); err != nil {
		return err
	}
	responses[name] = response
	return nil
}

func (r *responseBuilder) buildFromField(fld *types.Var, tpe types.Type, typable swaggerTypable, seen map[string]bool) error {
	debugLogf(r.ctx.debug, "build from field %s: %T", fld.Name(), tpe)

	switch ftpe := tpe.(type) {
	case *types.Basic:
		return swaggerSchemaForType(ftpe.Name(), typable)
	case *types.Struct:
		return r.buildFromFieldStruct(ftpe, typable)
	case *types.Pointer:
		return r.buildFromField(fld, ftpe.Elem(), typable, seen)
	case *types.Interface:
		return r.buildFromFieldInterface(ftpe, typable)
	case *types.Array:
		return r.buildFromField(fld, ftpe.Elem(), typable.Items(), seen)
	case *types.Slice:
		return r.buildFromField(fld, ftpe.Elem(), typable.Items(), seen)
	case *types.Map:
		return r.buildFromFieldMap(ftpe, typable)
	case *types.Named:
		return r.buildNamedField(ftpe, typable)
	case *types.Alias:
		debugLogf(r.ctx.debug, "alias(responses.buildFromField): got alias %v to %v", ftpe, ftpe.Rhs())
		return r.buildFieldAlias(ftpe, typable, fld, seen)
	default:
		return fmt.Errorf("unknown type for %s: %T: %w", fld.String(), fld.Type(), ErrCodeScan)
	}
}

func (r *responseBuilder) buildFromFieldStruct(ftpe *types.Struct, typable swaggerTypable) error {
	sb := schemaBuilder{
		decl: r.decl,
		ctx:  r.ctx,
	}

	if err := sb.buildFromType(ftpe, typable); err != nil {
		return err
	}

	r.postDecls = append(r.postDecls, sb.postDecls...)

	return nil
}

func (r *responseBuilder) buildFromFieldMap(ftpe *types.Map, typable swaggerTypable) error {
	schema := new(spec.Schema)
	typable.Schema().Typed("object", "").AdditionalProperties = &spec.SchemaOrBool{
		Schema: schema,
	}

	sb := schemaBuilder{
		decl: r.decl,
		ctx:  r.ctx,
	}

	if err := sb.buildFromType(ftpe.Elem(), schemaTypable{schema, typable.Level() + 1, r.ctx.opts.SkipExtensions}); err != nil {
		return err
	}

	r.postDecls = append(r.postDecls, sb.postDecls...)

	return nil
}

func (r *responseBuilder) buildFromFieldInterface(tpe types.Type, typable swaggerTypable) error {
	sb := schemaBuilder{
		decl: r.decl,
		ctx:  r.ctx,
	}
	if err := sb.buildFromType(tpe, typable); err != nil {
		return err
	}
	r.postDecls = append(r.postDecls, sb.postDecls...)

	return nil
}

func (r *responseBuilder) buildFromType(otpe types.Type, resp *spec.Response, seen map[string]bool) error {
	switch tpe := otpe.(type) {
	case *types.Pointer:
		return r.buildFromType(tpe.Elem(), resp, seen)
	case *types.Named:
		return r.buildNamedType(tpe, resp, seen)
	case *types.Alias:
		debugLogf(r.ctx.debug, "alias(responses.buildFromType): got alias %v to %v", tpe, tpe.Rhs())
		return r.buildAlias(tpe, resp, seen)
	default:
		return fmt.Errorf("anonymous types are currently not supported for responses: %w", ErrCodeScan)
	}
}

func (r *responseBuilder) buildNamedType(tpe *types.Named, resp *spec.Response, seen map[string]bool) error {
	o := tpe.Obj()
	if isAny(o) || isStdError(o) {
		return fmt.Errorf("%s type not supported in the context of a responses section definition: %w", o.Name(), ErrCodeScan)
	}
	mustNotBeABuiltinType(o)
	// ICI

	switch stpe := o.Type().Underlying().(type) { // TODO(fred): this is wrong without checking for aliases?
	case *types.Struct:
		debugLogf(r.ctx.debug, "build from type %s: %T", o.Name(), tpe)
		if decl, found := r.ctx.DeclForType(o.Type()); found {
			return r.buildFromStruct(decl, stpe, resp, seen)
		}
		return r.buildFromStruct(r.decl, stpe, resp, seen)

	default:
		if decl, found := r.ctx.DeclForType(o.Type()); found {
			var schema spec.Schema
			typable := schemaTypable{schema: &schema, level: 0, skipExt: r.ctx.opts.SkipExtensions}

			d := decl.Obj()
			if isStdTime(d) {
				typable.Typed("string", "date-time")
				return nil
			}
			if sfnm, isf := strfmtName(decl.Comments); isf {
				typable.Typed("string", sfnm)
				return nil
			}
			sb := &schemaBuilder{ctx: r.ctx, decl: decl}
			sb.inferNames()
			if err := sb.buildFromType(tpe.Underlying(), typable); err != nil {
				return err
			}
			resp.WithSchema(&schema)
			r.postDecls = append(r.postDecls, sb.postDecls...)
			return nil
		}
		return fmt.Errorf("responses can only be structs, did you mean for %s to be the response body?: %w", tpe.String(), ErrCodeScan)
	}
}

func (r *responseBuilder) buildAlias(tpe *types.Alias, resp *spec.Response, seen map[string]bool) error {
	// panic("yay")
	o := tpe.Obj()
	if isAny(o) || isStdError(o) {
		// wrong: TODO(fred): see what object exactly we want to build here - figure out with specific tests
		return fmt.Errorf("%s type not supported in the context of a responses section definition: %w", o.Name(), ErrCodeScan)
	}
	mustNotBeABuiltinType(o)
	mustHaveRightHandSide(tpe)

	rhs := tpe.Rhs()

	// If transparent aliases are enabled, use the underlying type directly without creating a definition
	if r.ctx.app.transparentAliases {
		return r.buildFromType(rhs, resp, seen)
	}

	decl, ok := r.ctx.FindModel(o.Pkg().Path(), o.Name())
	if !ok {
		return fmt.Errorf("can't find source file for aliased type: %v -> %v: %w", tpe, rhs, ErrCodeScan)
	}
	r.postDecls = append(r.postDecls, decl) // mark the left-hand side as discovered

	if !r.ctx.app.refAliases {
		// expand alias
		unaliased := types.Unalias(tpe)
		return r.buildFromType(unaliased.Underlying(), resp, seen)
	}

	switch rtpe := rhs.(type) {
	// load declaration for named unaliased type
	case *types.Named:
		o := rtpe.Obj()
		if o.Pkg() == nil {
			break // builtin
		}

		typable := schemaTypable{schema: &spec.Schema{}, level: 0, skipExt: r.ctx.opts.SkipExtensions}
		return r.makeRef(decl, typable)
	case *types.Alias:
		o := rtpe.Obj()
		if o.Pkg() == nil {
			break // builtin
		}

		typable := schemaTypable{schema: &spec.Schema{}, level: 0, skipExt: r.ctx.opts.SkipExtensions}

		return r.makeRef(decl, typable)
	}

	return r.buildFromType(rhs, resp, seen)
}

func (r *responseBuilder) buildNamedField(ftpe *types.Named, typable swaggerTypable) error {
	decl, found := r.ctx.DeclForType(ftpe.Obj().Type())
	if !found {
		return fmt.Errorf("unable to find package and source file for: %s: %w", ftpe.String(), ErrCodeScan)
	}

	d := decl.Obj()
	if isStdTime(d) {
		typable.Typed("string", "date-time")
		return nil
	}

	if sfnm, isf := strfmtName(decl.Comments); isf {
		typable.Typed("string", sfnm)
		return nil
	}

	sb := &schemaBuilder{ctx: r.ctx, decl: decl}
	sb.inferNames()
	if err := sb.buildFromType(decl.ObjType(), typable); err != nil {
		return err
	}

	r.postDecls = append(r.postDecls, sb.postDecls...)

	return nil
}

func (r *responseBuilder) buildFieldAlias(tpe *types.Alias, typable swaggerTypable, fld *types.Var, seen map[string]bool) error {
	_ = fld
	_ = seen
	o := tpe.Obj()
	if isAny(o) {
		// e.g. Field interface{} or Field any
		_ = typable.Schema()

		return nil // just leave an empty schema
	}

	// If transparent aliases are enabled, use the underlying type directly without creating a definition
	if r.ctx.app.transparentAliases {
		sb := schemaBuilder{
			decl: r.decl,
			ctx:  r.ctx,
		}
		if err := sb.buildFromType(tpe.Rhs(), typable); err != nil {
			return err
		}
		r.postDecls = append(r.postDecls, sb.postDecls...)
		return nil
	}

	decl, ok := r.ctx.FindModel(o.Pkg().Path(), o.Name())
	if !ok {
		return fmt.Errorf("can't find source file for aliased type: %v: %w", tpe, ErrCodeScan)
	}
	r.postDecls = append(r.postDecls, decl) // mark the left-hand side as discovered

	return r.makeRef(decl, typable)
}

func (r *responseBuilder) buildFromStruct(decl *entityDecl, tpe *types.Struct, resp *spec.Response, seen map[string]bool) error {
	if tpe.NumFields() == 0 {
		return nil
	}

	for fld := range tpe.Fields() {
		if fld.Embedded() {
			if err := r.buildFromType(fld.Type(), resp, seen); err != nil {
				return err
			}
			continue
		}
		if fld.Anonymous() {
			debugLogf(r.ctx.debug, "skipping anonymous field")
			continue
		}

		if err := r.processResponseField(fld, decl, resp, seen); err != nil {
			return err
		}
	}

	for k := range resp.Headers {
		if !seen[k] {
			delete(resp.Headers, k)
		}
	}
	return nil
}

func (r *responseBuilder) processResponseField(fld *types.Var, decl *entityDecl, resp *spec.Response, seen map[string]bool) error {
	if !fld.Exported() {
		return nil
	}

	afld := findASTField(decl.File, fld.Pos())
	if afld == nil {
		debugLogf(r.ctx.debug, "can't find source associated with %s", fld.String())
		return nil
	}

	if ignored(afld.Doc) {
		debugLogf(r.ctx.debug, "field %v is deliberately ignored", fld)
		return nil
	}

	name, ignore, _, _, err := parseJSONTag(afld)
	if err != nil {
		return err
	}
	if ignore {
		return nil
	}

	var in string
	// scan for param location first, this changes some behavior down the line
	if afld.Doc != nil {
		for _, cmt := range afld.Doc.List {
			for line := range strings.SplitSeq(cmt.Text, "\n") {
				matches := rxIn.FindStringSubmatch(line)
				if len(matches) > 0 && len(strings.TrimSpace(matches[1])) > 0 {
					in = strings.TrimSpace(matches[1])
				}
			}
		}
	}

	ps := resp.Headers[name]

	// support swagger:file for response
	// An API operation can return a file, such as an image or PDF. In this case,
	// define the response schema with type: file and specify the appropriate MIME types in the produces section.
	if afld.Doc != nil && fileParam(afld.Doc) {
		resp.Schema = &spec.Schema{}
		resp.Schema.Typed("file", "")
	} else {
		debugLogf(r.ctx.debug, "build response %v (%v) (not a file)", fld, fld.Type())
		if err := r.buildFromField(fld, fld.Type(), responseTypable{in, &ps, resp, r.ctx.opts.SkipExtensions}, seen); err != nil {
			return err
		}
	}

	if strfmtName, ok := strfmtName(afld.Doc); ok {
		ps.Typed("string", strfmtName)
	}

	sp := new(sectionedParser)
	sp.setDescription = func(lines []string) { ps.Description = joinDropLast(lines) }
	if err := setupResponseHeaderTaggers(sp, &ps, name, afld); err != nil {
		return err
	}

	if err := sp.Parse(afld.Doc); err != nil {
		return err
	}

	if in != bodyTag {
		seen[name] = true
		if resp.Headers == nil {
			resp.Headers = make(map[string]spec.Header)
		}
		resp.Headers[name] = ps
	}
	return nil
}

func (r *responseBuilder) makeRef(decl *entityDecl, prop swaggerTypable) error {
	nm, _ := decl.Names()
	ref, err := spec.NewRef("#/definitions/" + nm)
	if err != nil {
		return err
	}

	prop.SetRef(ref)
	r.postDecls = append(r.postDecls, decl) // mark the $ref target as discovered

	return nil
}
