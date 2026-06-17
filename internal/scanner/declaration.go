// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/go-openapi/codescan/internal/parsers"
	"golang.org/x/tools/go/packages"
)

type EntityDecl struct {
	Comments                *ast.CommentGroup
	Type                    *types.Named
	Alias                   *types.Alias // added to supplement Named, after go1.22
	Ident                   *ast.Ident
	Spec                    *ast.TypeSpec
	File                    *ast.File
	Pkg                     *packages.Package
	hasModelAnnotation      bool
	hasResponseAnnotation   bool
	hasParameterAnnotation  bool
	modelOverrideSuppressed bool
}

// SuppressModelOverride drops this declaration's `swagger:model <name>`
// override so that Names / DefKey fall back to the Go type name. Used to
// resolve a same-package duplicate, where two distinct types in one
// package claim the same override name (a user error): the first keeps
// the name, later ones revert to their Go name. See name-identity design
// D-4 (.claude/plans/name-identity-cyclic-ref.md §9.1).
func (d *EntityDecl) SuppressModelOverride() { d.modelOverrideSuppressed = true }

// ModelOverrideSuppressed reports whether SuppressModelOverride was set.
func (d *EntityDecl) ModelOverrideSuppressed() bool { return d.modelOverrideSuppressed }

// Obj returns the type name for the declaration defining the named type or alias t.
func (d *EntityDecl) Obj() *types.TypeName {
	if d.Type != nil {
		return d.Type.Obj()
	}
	if d.Alias != nil {
		return d.Alias.Obj()
	}

	panic("invalid EntityDecl: Type and Alias are both nil")
}

func (d *EntityDecl) ObjType() types.Type {
	if d.Type != nil {
		return d.Type
	}
	if d.Alias != nil {
		return d.Alias
	}

	panic("invalid EntityDecl: Type and Alias are both nil")
}

func (d *EntityDecl) Names() (name, goName string) {
	goName = d.Ident.Name
	model, ok := parsers.ModelOverride(d.Comments)
	if !ok {
		return goName, goName
	}

	d.hasModelAnnotation = true
	if model == "" || d.modelOverrideSuppressed {
		return goName, goName
	}

	return model, goName
}

// DefKey returns the fully-qualified, compiler-unique definition key for
// this declaration: "<pkgpath>/<name>", where <name> is the
// swagger:model override when present, else the Go type name (the first
// return of Names). This is the build-time key for the definitions map
// and for every "#/definitions/" $ref target, so two distinct Go types
// that share a short name can never collide before the spec.Builder's
// reduce stage shortens names back. See the name-identity / cyclic-$ref
// design (.claude/plans/name-identity-cyclic-ref.md §9.1, §12.1).
//
// Universe / package-less types (no enclosing package) fall back to the
// bare name; in practice those are intercepted as stdlib specials before
// they ever reach a definition key.
func (d *EntityDecl) DefKey() string {
	name, _ := d.Names()
	if pkg := d.Obj().Pkg(); pkg != nil {
		return pkg.Path() + "/" + name
	}
	return name
}

func (d *EntityDecl) ResponseNames() (name, goName string) {
	goName = d.Ident.Name
	response, ok := parsers.ResponseOverride(d.Comments)
	if !ok {
		return name, goName
	}

	d.hasResponseAnnotation = true
	if response == "" {
		return goName, goName
	}

	return response, goName
}

func (d *EntityDecl) OperationIDs() (result []string) {
	if d == nil {
		return nil
	}

	parameters, ok := parsers.ParametersOverride(d.Comments)
	if !ok {
		return nil
	}

	d.hasParameterAnnotation = true

	for _, parameter := range parameters {
		for param := range strings.SplitSeq(parameter, " ") {
			name := strings.TrimSpace(param)
			if len(name) > 0 {
				result = append(result, name)
			}
		}
	}

	return result
}

func (d *EntityDecl) HasModelAnnotation() bool {
	if d.hasModelAnnotation {
		return true
	}

	_, ok := parsers.ModelOverride(d.Comments)
	if !ok {
		return false
	}

	d.hasModelAnnotation = true

	return true
}

func (d *EntityDecl) HasResponseAnnotation() bool {
	if d.hasResponseAnnotation {
		return true
	}

	_, ok := parsers.ResponseOverride(d.Comments)
	if !ok {
		return false
	}

	d.hasResponseAnnotation = true

	return true
}

func (d *EntityDecl) HasParameterAnnotation() bool {
	if d.hasParameterAnnotation {
		return true
	}

	_, ok := parsers.ParametersOverride(d.Comments)
	if !ok {
		return false
	}

	d.hasParameterAnnotation = true

	return true
}
