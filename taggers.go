// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"fmt"
	"go/ast"

	"github.com/go-openapi/spec"
)

// itemsTaggers builds tag parsers for array items at a given nesting level.
func itemsTaggers(items *spec.Items, level int) []tagParser {
	// the expression is 1-index based not 0-index
	itemsPrefix := fmt.Sprintf(rxItemsPrefixFmt, level+1)

	return []tagParser{
		newSingleLineTagParser(fmt.Sprintf("items%dMaximum", level), &setMaximum{itemsValidations{items}, rxf(rxMaximumFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dMinimum", level), &setMinimum{itemsValidations{items}, rxf(rxMinimumFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dMultipleOf", level), &setMultipleOf{itemsValidations{items}, rxf(rxMultipleOfFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dMinLength", level), &setMinLength{itemsValidations{items}, rxf(rxMinLengthFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dMaxLength", level), &setMaxLength{itemsValidations{items}, rxf(rxMaxLengthFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dPattern", level), &setPattern{itemsValidations{items}, rxf(rxPatternFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dCollectionFormat", level), &setCollectionFormat{itemsValidations{items}, rxf(rxCollectionFormatFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dMinItems", level), &setMinItems{itemsValidations{items}, rxf(rxMinItemsFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dMaxItems", level), &setMaxItems{itemsValidations{items}, rxf(rxMaxItemsFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dUnique", level), &setUnique{itemsValidations{items}, rxf(rxUniqueFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dEnum", level), &setEnum{itemsValidations{items}, rxf(rxEnumFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dDefault", level), &setDefault{&items.SimpleSchema, itemsValidations{items}, rxf(rxDefaultFmt, itemsPrefix)}),
		newSingleLineTagParser(fmt.Sprintf("items%dExample", level), &setExample{&items.SimpleSchema, itemsValidations{items}, rxf(rxExampleFmt, itemsPrefix)}),
	}
}

// parseArrayTypes recursively builds tag parsers for nested array types.
func parseArrayTypes(sp *sectionedParser, name string, expr ast.Expr, items *spec.Items, level int) ([]tagParser, error) {
	if items == nil {
		return []tagParser{}, nil
	}
	switch iftpe := expr.(type) {
	case *ast.ArrayType:
		eleTaggers := itemsTaggers(items, level)
		sp.taggers = append(eleTaggers, sp.taggers...)
		return parseArrayTypes(sp, name, iftpe.Elt, items.Items, level+1)
	case *ast.SelectorExpr:
		return parseArrayTypes(sp, name, iftpe.Sel, items.Items, level+1)
	case *ast.Ident:
		taggers := []tagParser{}
		if iftpe.Obj == nil {
			taggers = itemsTaggers(items, level)
		}
		otherTaggers, err := parseArrayTypes(sp, name, expr, items.Items, level+1)
		if err != nil {
			return nil, err
		}
		return append(taggers, otherTaggers...), nil
	case *ast.StarExpr:
		return parseArrayTypes(sp, name, iftpe.X, items, level)
	default:
		return nil, fmt.Errorf("unknown field type ele for %q: %w", name, ErrCodeScan)
	}
}

// setupRefParamTaggers configures taggers for a parameter that is a $ref.
func setupRefParamTaggers(sp *sectionedParser, ps *spec.Parameter) {
	sp.taggers = []tagParser{
		newSingleLineTagParser("in", &matchOnlyParam{ps, rxIn}),
		newSingleLineTagParser("required", &matchOnlyParam{ps, rxRequired}),
		newMultiLineTagParser("Extensions", newSetExtensions(spExtensionsSetter(ps)), true),
	}
}

// setupInlineParamTaggers configures taggers for a fully-defined inline parameter.
func setupInlineParamTaggers(sp *sectionedParser, ps *spec.Parameter, name string, afld *ast.Field) error {
	sp.taggers = []tagParser{
		newSingleLineTagParser("in", &matchOnlyParam{ps, rxIn}),
		newSingleLineTagParser("maximum", &setMaximum{paramValidations{ps}, rxf(rxMaximumFmt, "")}),
		newSingleLineTagParser("minimum", &setMinimum{paramValidations{ps}, rxf(rxMinimumFmt, "")}),
		newSingleLineTagParser("multipleOf", &setMultipleOf{paramValidations{ps}, rxf(rxMultipleOfFmt, "")}),
		newSingleLineTagParser("minLength", &setMinLength{paramValidations{ps}, rxf(rxMinLengthFmt, "")}),
		newSingleLineTagParser("maxLength", &setMaxLength{paramValidations{ps}, rxf(rxMaxLengthFmt, "")}),
		newSingleLineTagParser("pattern", &setPattern{paramValidations{ps}, rxf(rxPatternFmt, "")}),
		newSingleLineTagParser("collectionFormat", &setCollectionFormat{paramValidations{ps}, rxf(rxCollectionFormatFmt, "")}),
		newSingleLineTagParser("minItems", &setMinItems{paramValidations{ps}, rxf(rxMinItemsFmt, "")}),
		newSingleLineTagParser("maxItems", &setMaxItems{paramValidations{ps}, rxf(rxMaxItemsFmt, "")}),
		newSingleLineTagParser("unique", &setUnique{paramValidations{ps}, rxf(rxUniqueFmt, "")}),
		newSingleLineTagParser("enum", &setEnum{paramValidations{ps}, rxf(rxEnumFmt, "")}),
		newSingleLineTagParser("default", &setDefault{&ps.SimpleSchema, paramValidations{ps}, rxf(rxDefaultFmt, "")}),
		newSingleLineTagParser("example", &setExample{&ps.SimpleSchema, paramValidations{ps}, rxf(rxExampleFmt, "")}),
		newSingleLineTagParser("required", &setRequiredParam{ps}),
		newMultiLineTagParser("Extensions", newSetExtensions(spExtensionsSetter(ps)), true),
	}

	// check if this is a primitive, if so parse the validations from the
	// doc comments of the slice declaration.
	if ftped, ok := afld.Type.(*ast.ArrayType); ok {
		taggers, err := parseArrayTypes(sp, name, ftped.Elt, ps.Items, 0)
		if err != nil {
			return err
		}
		sp.taggers = append(taggers, sp.taggers...)
	}

	return nil
}

// setupResponseHeaderTaggers configures taggers for a response header field.
func setupResponseHeaderTaggers(sp *sectionedParser, ps *spec.Header, name string, afld *ast.Field) error {
	sp.taggers = []tagParser{
		newSingleLineTagParser("maximum", &setMaximum{headerValidations{ps}, rxf(rxMaximumFmt, "")}),
		newSingleLineTagParser("minimum", &setMinimum{headerValidations{ps}, rxf(rxMinimumFmt, "")}),
		newSingleLineTagParser("multipleOf", &setMultipleOf{headerValidations{ps}, rxf(rxMultipleOfFmt, "")}),
		newSingleLineTagParser("minLength", &setMinLength{headerValidations{ps}, rxf(rxMinLengthFmt, "")}),
		newSingleLineTagParser("maxLength", &setMaxLength{headerValidations{ps}, rxf(rxMaxLengthFmt, "")}),
		newSingleLineTagParser("pattern", &setPattern{headerValidations{ps}, rxf(rxPatternFmt, "")}),
		newSingleLineTagParser("collectionFormat", &setCollectionFormat{headerValidations{ps}, rxf(rxCollectionFormatFmt, "")}),
		newSingleLineTagParser("minItems", &setMinItems{headerValidations{ps}, rxf(rxMinItemsFmt, "")}),
		newSingleLineTagParser("maxItems", &setMaxItems{headerValidations{ps}, rxf(rxMaxItemsFmt, "")}),
		newSingleLineTagParser("unique", &setUnique{headerValidations{ps}, rxf(rxUniqueFmt, "")}),
		newSingleLineTagParser("enum", &setEnum{headerValidations{ps}, rxf(rxEnumFmt, "")}),
		newSingleLineTagParser("default", &setDefault{&ps.SimpleSchema, headerValidations{ps}, rxf(rxDefaultFmt, "")}),
		newSingleLineTagParser("example", &setExample{&ps.SimpleSchema, headerValidations{ps}, rxf(rxExampleFmt, "")}),
	}

	// check if this is a primitive, if so parse the validations from the
	// doc comments of the slice declaration.
	if ftped, ok := afld.Type.(*ast.ArrayType); ok {
		taggers, err := parseArrayTypes(sp, name, ftped.Elt, ps.Items, 0)
		if err != nil {
			return err
		}
		sp.taggers = append(taggers, sp.taggers...)
	}

	return nil
}
