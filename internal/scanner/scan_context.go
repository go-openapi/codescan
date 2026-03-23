// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"iter"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/go-openapi/codescan/internal/logger"
	"github.com/go-openapi/codescan/internal/parsers"
	"golang.org/x/tools/go/packages"
)

const pkgLoadMode = packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

type node uint32

const (
	metaNode node = 1 << iota
	routeNode
	operationNode
	modelNode
	parametersNode
	responseNode
)

type ScanCtx struct {
	pkgs  []*packages.Package
	app   *TypeIndex
	debug bool

	opts *Options
}

func NewScanCtx(opts *Options) (*ScanCtx, error) {
	cfg := &packages.Config{
		Dir:   opts.WorkDir,
		Mode:  pkgLoadMode,
		Tests: false,
	}
	if opts.BuildTags != "" {
		cfg.BuildFlags = []string{"-tags", opts.BuildTags}
	}

	pkgs, err := packages.Load(cfg, opts.Packages...)
	if err != nil {
		return nil, err
	}

	app, err := NewTypeIndex(pkgs,
		WithExcludeDeps(opts.ExcludeDeps),
		WithIncludeTags(sliceToSet(opts.IncludeTags)),
		WithExcludeTags(sliceToSet(opts.ExcludeTags)),
		WithIncludePkgs(opts.Include),
		WithExcludePkgs(opts.Exclude),
		WithXNullableForPointers(opts.SetXNullableForPointers),
		WithRefAliases(opts.RefAliases),
		WithTransparentAliases(opts.TransparentAliases),
		WithDebug(opts.Debug),
	)
	if err != nil {
		return nil, err
	}

	return &ScanCtx{
		pkgs:  pkgs,
		app:   app,
		debug: opts.Debug,
		opts:  opts,
	}, nil
}

func (s *ScanCtx) SkipExtensions() bool {
	return s.opts.SkipExtensions
}

func (s *ScanCtx) DescWithRef() bool {
	return s.opts.DescWithRef
}

func (s *ScanCtx) SetXNullableForPointers() bool {
	return s.opts.SetXNullableForPointers
}

func (s *ScanCtx) TransparentAliases() bool {
	return s.opts.TransparentAliases
}

func (s *ScanCtx) RefAliases() bool {
	return s.opts.RefAliases
}

func (s *ScanCtx) Debug() bool {
	return s.debug
}

func (s *ScanCtx) Meta() iter.Seq[parsers.MetaSection] {
	if s.app == nil {
		return nil
	}

	return slices.Values(s.app.Meta)
}

func (s *ScanCtx) Operations() iter.Seq[parsers.ParsedPathContent] {
	if s.app == nil {
		return nil
	}

	return slices.Values(s.app.Operations)
}

func (s *ScanCtx) Routes() iter.Seq[parsers.ParsedPathContent] {
	if s.app == nil {
		return nil
	}

	return slices.Values(s.app.Routes)
}

func (s *ScanCtx) Responses() iter.Seq[*EntityDecl] {
	if s.app == nil {
		return nil
	}

	return slices.Values(s.app.Responses)
}

func (s *ScanCtx) Parameters() iter.Seq[*EntityDecl] {
	if s.app == nil {
		return nil
	}

	return slices.Values(s.app.Parameters)
}

func (s *ScanCtx) Models() iter.Seq2[*ast.Ident, *EntityDecl] {
	if s.app == nil {
		return nil
	}

	return maps.All(s.app.Models)
}

func (s *ScanCtx) NumExtraModels() int {
	if s.app == nil {
		return 0
	}

	return len(s.app.ExtraModels)
}

func (s *ScanCtx) ExtraModels() iter.Seq2[*ast.Ident, *EntityDecl] {
	if s.app == nil {
		return nil
	}

	return maps.All(s.app.ExtraModels)
}

func (s *ScanCtx) MoveExtraToModel(k *ast.Ident) {
	v, ok := s.app.ExtraModels[k]
	if !ok {
		return
	}

	s.app.Models[k] = v
	delete(s.app.ExtraModels, k)
}

func (s *ScanCtx) FindDecl(pkgPath, name string) (*EntityDecl, bool) {
	pkg, ok := s.app.AllPackages[pkgPath]
	if !ok {
		return nil, false
	}

	for _, file := range pkg.Syntax {
		for _, d := range file.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, sp := range gd.Specs {
				ts, ok := sp.(*ast.TypeSpec)
				if !ok || ts.Name.Name != name {
					continue
				}

				def, ok := pkg.TypesInfo.Defs[ts.Name]
				if !ok {
					logger.DebugLogf(s.debug, "couldn't find type info for %s", ts.Name)
					continue
				}

				nt, isNamed := def.Type().(*types.Named)
				at, isAliased := def.Type().(*types.Alias)
				if !isNamed && !isAliased {
					logger.DebugLogf(s.debug, "%s is not a named or an aliased type but a %T", ts.Name, def.Type())
					continue
				}

				comments := ts.Doc // type ( /* doc */ Foo struct{} )
				if comments == nil {
					comments = gd.Doc // /* doc */  type ( Foo struct{} )
				}

				return &EntityDecl{
					Comments: comments,
					Type:     nt,
					Alias:    at,
					Ident:    ts.Name,
					Spec:     ts,
					File:     file,
					Pkg:      pkg,
				}, true
			}
		}
	}

	return nil, false
}

func (s *ScanCtx) FindModel(pkgPath, name string) (*EntityDecl, bool) {
	for _, cand := range s.app.Models {
		ct := cand.Obj()
		if ct.Name() == name && ct.Pkg().Path() == pkgPath {
			return cand, true
		}
	}

	if decl, found := s.FindDecl(pkgPath, name); found {
		s.app.ExtraModels[decl.Ident] = decl
		return decl, true
	}

	return nil, false
}

func (s *ScanCtx) DeclForType(t types.Type) (*EntityDecl, bool) {
	switch tpe := t.(type) {
	case *types.Pointer:
		return s.DeclForType(tpe.Elem())
	case *types.Named:
		return s.FindDecl(tpe.Obj().Pkg().Path(), tpe.Obj().Name())
	case *types.Alias:
		return s.FindDecl(tpe.Obj().Pkg().Path(), tpe.Obj().Name())
	default:
		log.Printf("WARNING: unknown type to find the package for [%T]: %s", t, t.String())

		return nil, false
	}
}

func (s *ScanCtx) PkgForType(t types.Type) (*packages.Package, bool) {
	switch tpe := t.(type) {
	// case *types.Basic:
	// case *types.Struct:
	// case *types.Pointer:
	// case *types.Interface:
	// case *types.Array:
	// case *types.Slice:
	// case *types.Map:
	case *types.Named:
		v, ok := s.app.AllPackages[tpe.Obj().Pkg().Path()]
		return v, ok
	case *types.Alias:
		v, ok := s.app.AllPackages[tpe.Obj().Pkg().Path()]
		return v, ok
	default:
		log.Printf("WARNING: unknown type to find the package for [%T]: %s", t, t.String())
		return nil, false
	}
}

func (s *ScanCtx) FindComments(pkg *packages.Package, name string) (*ast.CommentGroup, bool) {
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, s := range gd.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok {
					if ts.Name.Name == name {
						return gd.Doc, true
					}
				}
			}
		}
	}
	return nil, false
}

func (s *ScanCtx) FindEnumValues(pkg *packages.Package, enumName string) (list []any, descList []string, _ bool) {
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}

			if gd.Tok != token.CONST {
				continue
			}

			for _, spec := range gd.Specs {
				literalValue, description := s.findEnumValue(spec, enumName)
				if literalValue == nil {
					continue
				}

				list = append(list, literalValue)
				descList = append(descList, description)
			}
		}
	}

	return list, descList, true
}

func (s *ScanCtx) findEnumValue(spec ast.Spec, enumName string) (literalValue any, description string) {
	vs, ok := spec.(*ast.ValueSpec)
	if !ok {
		return nil, ""
	}

	vsIdent, ok := vs.Type.(*ast.Ident)
	if !ok {
		return nil, ""
	}

	if vsIdent.Name != enumName {
		return nil, ""
	}

	if len(vs.Values) == 0 {
		return nil, ""
	}

	bl, ok := vs.Values[0].(*ast.BasicLit)
	if !ok {
		return nil, ""
	}

	literalValue = parsers.GetEnumBasicLitValue(bl)

	// build the enum description
	var (
		desc     = &strings.Builder{}
		namesLen = len(vs.Names)
	)

	fmt.Fprintf(desc, "%v ", literalValue)
	for i, name := range vs.Names {
		desc.WriteString(name.Name)
		if i < namesLen-1 {
			desc.WriteString(" ")
		}
	}

	if vs.Doc != nil {
		docListLen := len(vs.Doc.List)
		if docListLen > 0 {
			desc.WriteString(" ")
		}

		for i, doc := range vs.Doc.List {
			if doc.Text != "" {
				text := strings.TrimPrefix(doc.Text, "//")
				desc.WriteString(text)
				if i < docListLen-1 {
					desc.WriteString(" ")
				}
			}
		}
	}

	description = desc.String()

	return literalValue, description
}

func sliceToSet(names []string) map[string]bool {
	result := make(map[string]bool)
	for _, v := range names {
		result[v] = true
	}
	return result
}
