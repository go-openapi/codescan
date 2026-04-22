// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package routes

import "github.com/go-openapi/spec"

func opResponsesSetter(op *spec.Operation) func(*spec.Response, map[int]spec.Response) {
	return func(def *spec.Response, scr map[int]spec.Response) {
		if op.Responses == nil {
			op.Responses = new(spec.Responses)
		}
		op.Responses.Default = def
		op.Responses.StatusCodeResponses = scr
	}
}

func opParamSetter(op *spec.Operation) func([]*spec.Parameter) {
	return func(params []*spec.Parameter) {
		for _, v := range params {
			op.AddParam(v)
		}
	}
}

func opExtensionsSetter(op *spec.Operation) func(*spec.Extensions) {
	return func(exts *spec.Extensions) {
		for name, value := range *exts {
			op.AddExtension(name, value)
		}
	}
}
