// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag/conv"

	"golang.org/x/tools/go/packages"
)

const pkgLoadMode = packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

func safeConvert(str string) bool {
	b, err := conv.ConvertBool(str)
	if err != nil {
		return false
	}
	return b
}

// Debug is true when process is run with DEBUG=1 env var.
var Debug = safeConvert(os.Getenv("DEBUG")) //nolint:gochecknoglobals // package-level configuration from environment

type node uint32

const (
	metaNode node = 1 << iota
	routeNode
	operationNode
	modelNode
	parametersNode
	responseNode
)

// Options for the scanner.
type Options struct {
	Packages                []string
	InputSpec               *spec.Swagger
	ScanModels              bool
	WorkDir                 string
	BuildTags               string
	ExcludeDeps             bool
	Include                 []string
	Exclude                 []string
	IncludeTags             []string
	ExcludeTags             []string
	SetXNullableForPointers bool
	RefAliases              bool // aliases result in $ref, otherwise aliases are expanded
	TransparentAliases      bool // aliases are completely transparent, never creating definitions
	DescWithRef             bool // allow overloaded descriptions together with $ref, otherwise jsonschema draft4 $ref predates everything
	SkipExtensions          bool // skip generating x-go-* vendor extensions in the spec
}

type scanCtx struct {
	pkgs []*packages.Package
	app  *typeIndex

	opts *Options
}

func sliceToSet(names []string) map[string]bool {
	result := make(map[string]bool)
	for _, v := range names {
		result[v] = true
	}
	return result
}

// Run the scanner to produce a spec with the options provided.
func Run(opts *Options) (*spec.Swagger, error) {
	sc, err := newScanCtx(opts)
	if err != nil {
		return nil, err
	}
	sb := newSpecBuilder(opts.InputSpec, sc, opts.ScanModels)
	return sb.Build()
}

func newScanCtx(opts *Options) (*scanCtx, error) {
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

	app, err := newTypeIndex(pkgs,
		withExcludeDeps(opts.ExcludeDeps),
		withIncludeTags(sliceToSet(opts.IncludeTags)),
		withExcludeTags(sliceToSet(opts.ExcludeTags)),
		withIncludePkgs(opts.Include),
		withExcludePkgs(opts.Exclude),
		withXNullableForPointers(opts.SetXNullableForPointers),
		withRefAliases(opts.RefAliases),
		withTransparentAliases(opts.TransparentAliases),
	)
	if err != nil {
		return nil, err
	}

	return &scanCtx{
		pkgs: pkgs,
		app:  app,
		opts: opts,
	}, nil
}

type entityDecl struct {
	Comments               *ast.CommentGroup
	Type                   *types.Named
	Alias                  *types.Alias // added to supplement Named, after go1.22
	Ident                  *ast.Ident
	Spec                   *ast.TypeSpec
	File                   *ast.File
	Pkg                    *packages.Package
	hasModelAnnotation     bool
	hasResponseAnnotation  bool
	hasParameterAnnotation bool
}

// Obj returns the type name for the declaration defining the named type or alias t.
func (d *entityDecl) Obj() *types.TypeName {
	if d.Type != nil {
		return d.Type.Obj()
	}
	if d.Alias != nil {
		return d.Alias.Obj()
	}

	panic("invalid entityDecl: Type and Alias are both nil")
}

func (d *entityDecl) ObjType() types.Type {
	if d.Type != nil {
		return d.Type
	}
	if d.Alias != nil {
		return d.Alias
	}

	panic("invalid entityDecl: Type and Alias are both nil")
}

func (d *entityDecl) Names() (name, goName string) {
	goName = d.Ident.Name
	name = goName
	if d.Comments == nil {
		return name, goName
	}

DECLS:
	for _, cmt := range d.Comments.List {
		for ln := range strings.SplitSeq(cmt.Text, "\n") {
			matches := rxModelOverride.FindStringSubmatch(ln)
			if len(matches) > 0 {
				d.hasModelAnnotation = true
			}
			if len(matches) > 1 && len(matches[1]) > 0 {
				name = matches[1]
				break DECLS
			}
		}
	}

	return name, goName
}

func (d *entityDecl) ResponseNames() (name, goName string) {
	goName = d.Ident.Name
	name = goName
	if d.Comments == nil {
		return name, goName
	}

DECLS:
	for _, cmt := range d.Comments.List {
		for ln := range strings.SplitSeq(cmt.Text, "\n") {
			matches := rxResponseOverride.FindStringSubmatch(ln)
			if len(matches) > 0 {
				d.hasResponseAnnotation = true
			}
			if len(matches) > 1 && len(matches[1]) > 0 {
				name = matches[1]
				break DECLS
			}
		}
	}
	return name, goName
}

func (d *entityDecl) OperationIDs() (result []string) {
	if d == nil || d.Comments == nil {
		return nil
	}

	for _, cmt := range d.Comments.List {
		for ln := range strings.SplitSeq(cmt.Text, "\n") {
			matches := rxParametersOverride.FindStringSubmatch(ln)
			if len(matches) > 0 {
				d.hasParameterAnnotation = true
			}
			if len(matches) > 1 && len(matches[1]) > 0 {
				for pt := range strings.SplitSeq(matches[1], " ") {
					tr := strings.TrimSpace(pt)
					if len(tr) > 0 {
						result = append(result, tr)
					}
				}
			}
		}
	}
	return result
}

func (d *entityDecl) HasModelAnnotation() bool {
	if d.hasModelAnnotation {
		return true
	}
	if d.Comments == nil {
		return false
	}
	for _, cmt := range d.Comments.List {
		for ln := range strings.SplitSeq(cmt.Text, "\n") {
			matches := rxModelOverride.FindStringSubmatch(ln)
			if len(matches) > 0 {
				d.hasModelAnnotation = true
				return true
			}
		}
	}
	return false
}

func (d *entityDecl) HasResponseAnnotation() bool {
	if d.hasResponseAnnotation {
		return true
	}
	if d.Comments == nil {
		return false
	}
	for _, cmt := range d.Comments.List {
		for ln := range strings.SplitSeq(cmt.Text, "\n") {
			matches := rxResponseOverride.FindStringSubmatch(ln)
			if len(matches) > 0 {
				d.hasResponseAnnotation = true
				return true
			}
		}
	}
	return false
}

func (d *entityDecl) HasParameterAnnotation() bool {
	if d.hasParameterAnnotation {
		return true
	}
	if d.Comments == nil {
		return false
	}
	for _, cmt := range d.Comments.List {
		for ln := range strings.SplitSeq(cmt.Text, "\n") {
			matches := rxParametersOverride.FindStringSubmatch(ln)
			if len(matches) > 0 {
				d.hasParameterAnnotation = true
				return true
			}
		}
	}
	return false
}

func (s *scanCtx) FindDecl(pkgPath, name string) (*entityDecl, bool) {
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
					debugLogf("couldn't find type info for %s", ts.Name)
					continue
				}

				nt, isNamed := def.Type().(*types.Named)
				at, isAliased := def.Type().(*types.Alias)
				if !isNamed && !isAliased {
					debugLogf("%s is not a named or an aliased type but a %T", ts.Name, def.Type())
					continue
				}

				comments := ts.Doc // type ( /* doc */ Foo struct{} )
				if comments == nil {
					comments = gd.Doc // /* doc */  type ( Foo struct{} )
				}

				return &entityDecl{
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

func (s *scanCtx) FindModel(pkgPath, name string) (*entityDecl, bool) {
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

func (s *scanCtx) PkgForPath(pkgPath string) (*packages.Package, bool) {
	v, ok := s.app.AllPackages[pkgPath]
	return v, ok
}

func (s *scanCtx) DeclForType(t types.Type) (*entityDecl, bool) {
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

func (s *scanCtx) PkgForType(t types.Type) (*packages.Package, bool) {
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
		log.Printf("unknown type to find the package for [%T]: %s", t, t.String())
		return nil, false
	}
}

func (s *scanCtx) FindComments(pkg *packages.Package, name string) (*ast.CommentGroup, bool) {
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

func (s *scanCtx) FindEnumValues(pkg *packages.Package, enumName string) (list []any, descList []string, _ bool) {
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

func (s *scanCtx) findEnumValue(spec ast.Spec, enumName string) (literalValue any, description string) {
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

	literalValue = getEnumBasicLitValue(bl)

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

type typeIndexOption func(*typeIndex)

func withExcludeDeps(excluded bool) typeIndexOption {
	return func(a *typeIndex) {
		a.excludeDeps = excluded
	}
}

func withIncludeTags(included map[string]bool) typeIndexOption {
	return func(a *typeIndex) {
		a.includeTags = included
	}
}

func withExcludeTags(excluded map[string]bool) typeIndexOption {
	return func(a *typeIndex) {
		a.excludeTags = excluded
	}
}

func withIncludePkgs(included []string) typeIndexOption {
	return func(a *typeIndex) {
		a.includePkgs = included
	}
}

func withExcludePkgs(excluded []string) typeIndexOption {
	return func(a *typeIndex) {
		a.excludePkgs = excluded
	}
}

func withXNullableForPointers(enabled bool) typeIndexOption {
	return func(a *typeIndex) {
		a.setXNullableForPointers = enabled
	}
}

func withRefAliases(enabled bool) typeIndexOption {
	return func(a *typeIndex) {
		a.refAliases = enabled
	}
}

func withTransparentAliases(enabled bool) typeIndexOption {
	return func(a *typeIndex) {
		a.transparentAliases = enabled
	}
}

func newTypeIndex(pkgs []*packages.Package, opts ...typeIndexOption) (*typeIndex, error) {
	ac := &typeIndex{
		AllPackages: make(map[string]*packages.Package),
		Models:      make(map[*ast.Ident]*entityDecl),
		ExtraModels: make(map[*ast.Ident]*entityDecl),
	}
	for _, apply := range opts {
		apply(ac)
	}

	if err := ac.build(pkgs); err != nil {
		return nil, err
	}
	return ac, nil
}

type typeIndex struct {
	AllPackages             map[string]*packages.Package
	Models                  map[*ast.Ident]*entityDecl
	ExtraModels             map[*ast.Ident]*entityDecl
	Meta                    []metaSection
	Routes                  []parsedPathContent
	Operations              []parsedPathContent
	Parameters              []*entityDecl
	Responses               []*entityDecl
	excludeDeps             bool
	includeTags             map[string]bool
	excludeTags             map[string]bool
	includePkgs             []string
	excludePkgs             []string
	setXNullableForPointers bool
	refAliases              bool
	transparentAliases      bool
}

func (a *typeIndex) build(pkgs []*packages.Package) error {
	for _, pkg := range pkgs {
		if _, known := a.AllPackages[pkg.PkgPath]; known {
			continue
		}
		a.AllPackages[pkg.PkgPath] = pkg
		if err := a.processPackage(pkg); err != nil {
			return err
		}
		if err := a.walkImports(pkg); err != nil {
			return err
		}
	}

	return nil
}

func (a *typeIndex) processPackage(pkg *packages.Package) error {
	if !shouldAcceptPkg(pkg.PkgPath, a.includePkgs, a.excludePkgs) {
		debugLogf("package %s is ignored due to rules", pkg.Name)
		return nil
	}

	for _, file := range pkg.Syntax {
		if err := a.processFile(pkg, file); err != nil {
			return err
		}
	}

	return nil
}

func (a *typeIndex) processFile(pkg *packages.Package, file *ast.File) error {
	n, err := a.detectNodes(file)
	if err != nil {
		return err
	}

	if n&metaNode != 0 {
		a.Meta = append(a.Meta, metaSection{Comments: file.Doc})
	}

	if n&operationNode != 0 {
		a.Operations = a.collectPathAnnotations(rxOperation, file.Comments, a.Operations)
	}

	if n&routeNode != 0 {
		a.Routes = a.collectPathAnnotations(rxRoute, file.Comments, a.Routes)
	}

	a.processFileDecls(pkg, file, n)

	return nil
}

func (a *typeIndex) collectPathAnnotations(rx *regexp.Regexp, comments []*ast.CommentGroup, dst []parsedPathContent) []parsedPathContent {
	for _, cmts := range comments {
		pp := parsePathAnnotation(rx, cmts.List)
		if pp.Method == "" {
			continue
		}
		if !shouldAcceptTag(pp.Tags, a.includeTags, a.excludeTags) {
			debugLogf("operation %s %s is ignored due to tag rules", pp.Method, pp.Path)
			continue
		}
		dst = append(dst, pp)
	}
	return dst
}

func (a *typeIndex) processFileDecls(pkg *packages.Package, file *ast.File, n node) {
	for _, dt := range file.Decls {
		switch fd := dt.(type) {
		case *ast.BadDecl:
			continue
		case *ast.FuncDecl:
			if fd.Body == nil {
				continue
			}
			for _, stmt := range fd.Body.List {
				if dstm, ok := stmt.(*ast.DeclStmt); ok {
					if gd, isGD := dstm.Decl.(*ast.GenDecl); isGD {
						a.processDecl(pkg, file, n, gd)
					}
				}
			}
		case *ast.GenDecl:
			a.processDecl(pkg, file, n, fd)
		}
	}
}

func (a *typeIndex) processDecl(pkg *packages.Package, file *ast.File, n node, gd *ast.GenDecl) {
	for _, sp := range gd.Specs {
		switch ts := sp.(type) {
		case *ast.ValueSpec:
			debugLogf("saw value spec: %v", ts.Names)
			return
		case *ast.ImportSpec:
			debugLogf("saw import spec: %v", ts.Name)
			return
		case *ast.TypeSpec:
			def, ok := pkg.TypesInfo.Defs[ts.Name]
			if !ok {
				debugLogf("couldn't find type info for %s", ts.Name)
				continue
			}
			nt, isNamed := def.Type().(*types.Named)
			at, isAliased := def.Type().(*types.Alias)
			if !isNamed && !isAliased {
				debugLogf("%s is not a named or aliased type but a %T", ts.Name, def.Type())

				continue
			}

			comments := ts.Doc // type ( /* doc */ Foo struct{} )
			if comments == nil {
				comments = gd.Doc // /* doc */  type ( Foo struct{} )
			}

			decl := &entityDecl{
				Comments: comments,
				Type:     nt,
				Alias:    at,
				Ident:    ts.Name,
				Spec:     ts,
				File:     file,
				Pkg:      pkg,
			}
			key := ts.Name
			switch {
			case n&modelNode != 0 && decl.HasModelAnnotation():
				a.Models[key] = decl
			case n&parametersNode != 0 && decl.HasParameterAnnotation():
				a.Parameters = append(a.Parameters, decl)
			case n&responseNode != 0 && decl.HasResponseAnnotation():
				a.Responses = append(a.Responses, decl)
			default:
				debugLogf(
					"type %q skipped because it is not tagged as a model, a parameter or a response. %s",
					decl.Obj().Name(),
					"It may reenter the scope because it is a discovered dependency",
				)
			}
		}
	}
}

func (a *typeIndex) walkImports(pkg *packages.Package) error {
	if a.excludeDeps {
		return nil
	}
	for _, v := range pkg.Imports {
		if _, known := a.AllPackages[v.PkgPath]; known {
			continue
		}

		a.AllPackages[v.PkgPath] = v
		if err := a.processPackage(v); err != nil {
			return err
		}
		if err := a.walkImports(v); err != nil {
			return err
		}
	}
	return nil
}

func checkStructConflict(seenStruct *string, annotation string, text string) error {
	if *seenStruct != "" && *seenStruct != annotation {
		return fmt.Errorf("classifier: already annotated as %s, can't also be %q - %s: %w", *seenStruct, annotation, text, ErrCodeScan)
	}
	*seenStruct = annotation
	return nil
}

// detectNodes scans all comment groups in a file and returns a bitmask of
// detected swagger annotation types. Node types like route, operation, and
// meta accumulate freely across comment groups. Struct-level annotations
// (model, parameters, response) are mutually exclusive within a single
// comment group — mixing them is an error.
func (a *typeIndex) detectNodes(file *ast.File) (node, error) {
	var n node
	for _, comments := range file.Comments {
		var seenStruct string // tracks the struct annotation for this comment group
		for _, cline := range comments.List {
			if cline == nil {
				continue
			}
		}

		for _, cline := range comments.List {
			if cline == nil {
				continue
			}

			matches := rxSwaggerAnnotation.FindStringSubmatch(cline.Text)
			if len(matches) < minAnnotationMatch {
				continue
			}

			switch matches[1] {
			case "route":
				n |= routeNode
			case "operation":
				n |= operationNode
			case "model":
				n |= modelNode
				if err := checkStructConflict(&seenStruct, matches[1], cline.Text); err != nil {
					return 0, err
				}
			case "meta":
				n |= metaNode
			case "parameters":
				n |= parametersNode
				if err := checkStructConflict(&seenStruct, matches[1], cline.Text); err != nil {
					return 0, err
				}
			case "response":
				n |= responseNode
				if err := checkStructConflict(&seenStruct, matches[1], cline.Text); err != nil {
					return 0, err
				}
			case "strfmt", paramNameKey, "discriminated", "file", "enum", "default", "alias", "type":
				// TODO: perhaps collect these and pass along to avoid lookups later on
			case "allOf":
			case "ignore":
			default:
				return 0, fmt.Errorf("classifier: unknown swagger annotation %q: %w", matches[1], ErrCodeScan)
			}
		}
	}

	return n, nil
}

func debugLogf(format string, args ...any) {
	if Debug {
		_ = log.Output(logCallerDepth, fmt.Sprintf(format, args...))
	}
}
