// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1708

// ListResponseMeta is a nested model.
//
// swagger:model
type ListResponseMeta struct {
	Total int64 `json:"total"`
}

// ListResponseData is a nested model.
//
// swagger:model
type ListResponseData struct {
	Items []string `json:"items"`
}

// EventsResponse is a body response whose fields are pointers to other models —
// the reporter hit "missing property type" on this shape (follow-up of #619).
//
// swagger:response eventsResponse
type EventsResponse struct {
	// in: body
	Body struct {
		Meta *ListResponseMeta `json:"meta"`
		Data *ListResponseData `json:"data"`
	}
}
