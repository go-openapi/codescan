// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package name_from_tags witnesses the NameFromTags option: the emitted
// name of a schema property, query parameter, or response header is
// sourced from the first struct-tag type listed in Options.NameFromTags.
//
// Every field below carries BOTH a json: and a form: tag whose names
// differ. Under the default (nil → ["json"]) the json name is used;
// under ["form","json"] the form name wins; under an explicit empty
// slice the Go field name is used. The encoding/json directives
// (`,omitempty`, `json:"-"`) are always read from the json tag,
// independent of the setting.
package name_from_tags

// Filter is a model whose fields carry differing json: and form: names.
//
// swagger:model Filter
type Filter struct {
	// SortKey is named "sortKey" by default, "sort_key" under form-first.
	SortKey string `json:"sortKey" form:"sort_key"`

	// PageSize keeps its json omitempty directive whichever tag names it.
	PageSize int `json:"pageSize,omitempty" form:"page_size"`

	// Label has only a json tag — same name under every setting.
	Label string `json:"label" form:"-"`

	// Internal is excluded by json:"-" regardless of NameFromTags, even
	// though the form tag would name it.
	Internal string `json:"-" form:"internal"`
}

// ListParams are the query parameters for the list operation.
//
// swagger:parameters listItems
type ListParams struct {
	// SortKey selects the sort column.
	//
	// in: query
	SortKey string `json:"sortKey" form:"sort_key"`

	// PageSize bounds the page length.
	//
	// in: query
	PageSize int `json:"pageSize" form:"page_size"`
}

// ListResponse carries response headers with differing json: and form: names.
//
// swagger:response listResponse
type ListResponse struct {
	// RequestID is a correlation header.
	//
	// in: header
	RequestID string `json:"X-Request-Id" form:"x_request_id"`

	// Body is the payload.
	//
	// in: body
	Body struct {
		// Items is the page of matching filters.
		Items []Filter `json:"items"`
	} `json:"body"`
}

// ListItems swagger:route GET /items items listItems
//
// List items.
//
// Responses:
//
//	200: listResponse
func ListItems() {}
