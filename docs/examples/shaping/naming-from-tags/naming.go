// SPDX-License-Identifier: Apache-2.0

// Package namingfromtags holds the annotated declaration used by the
// "Naming from struct tags" how-to. naming_test.go scans it twice — with
// the default NameFromTags (["json"]) and with ["form","json"] — and writes
// the before/after golden fragments the guide renders.
package namingfromtags

// snippet:model

// Filter is a query model whose fields carry both json: and form: tags.
//
// swagger:model
type Filter struct {
	// SortKey selects the sort column.
	SortKey string `form:"sort_key" json:"sortKey"`

	// PageSize bounds the page length.
	PageSize int `form:"page_size" json:"pageSize"`
}

// endsnippet:model
