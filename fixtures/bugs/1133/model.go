// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1133

// MyModel is the only annotated model.
//
// swagger:model foo
type MyModel struct {
	Name string `json:"name"`
}

// TranslateFunc is an unsupported (function) type, unrelated to the spec — like
// the imported go-i18n type that used to halt the whole scan (#1133).
type TranslateFunc func(translationID string, args ...interface{}) string

// SomeStruct is a non-annotated struct that uses the function type (#1174).
type SomeStruct struct {
	Fn TranslateFunc
}
