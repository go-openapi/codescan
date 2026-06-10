// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build testscanner

package schema

// UUID should be discovered through dependency analysis.
//
// Pre-R6 this comment held: the (now-retired) discovery loop would
// pull UUID into `definitions` as a side effect of being referenced
// by `order.ID`. Post-R6 the unannotated alias dissolves at use
// sites; see UUIDModeled below for the explicit-opt-in counterpart.
type UUID = int64

// Anything should be discovered through dependency analysis.
type Anything = any

// Empty should be discovered through dependency analysis.
type Empty = struct{}

// UUIDModeled is the annotated counterpart of `UUID`. Same Go
// type, but the `swagger:model` annotation makes it a first-class
// spec entity: it surfaces in `definitions`, and any field typed
// `UUIDModeled` produces `$ref: #/definitions/UUIDModeled` rather
// than dissolving to the underlying primitive shape.
//
// swagger:model UUIDModeled
type UUIDModeled = int64

// AnythingModeled is the annotated counterpart of `Anything`. The
// `swagger:model` annotation preserves the alias identity at use
// sites despite the open `any` underlying type.
//
// swagger:model AnythingModeled
type AnythingModeled = any

// # StoreOrder represents an order in this application.
//
// An order can either be created, processed or completed.
//
// swagger:model order
type StoreOrder struct {
	// the id for this order
	//
	// required: true
	// min: 1
	ID UUID `json:"id"`

	EID ExtendedID `json:"extended_id"`

	// the name for this user
	//
	// required: true
	// min length: 3
	UserID int64 `json:"userId"`

	// the category of this user
	//
	// required: true
	// default: bar
	// enum: foo,bar,none
	Category string `json:"category"`

	// the items for this order
	Items []struct {
		ID           int32 `json:"id"`
		Quantity     int16 `json:"quantity"`
		ExtraOptions any   `json:"extra_options"`
	} `json:"items"`

	Extras any

	MoreExtras     interface{}
	DeliveryOption Anything
}

// StoreOrderModeled is the bidirectional sibling of StoreOrder.
// Same field layout, but the alias-typed fields use the ANNOTATED
// aliases (UUIDModeled / AnythingModeled). Per R6, each annotated
// alias preserves its identity at the field site:
//
//   - ID UUIDModeled               → $ref: #/definitions/UUIDModeled
//   - DeliveryOption AnythingModeled → $ref: #/definitions/AnythingModeled
//   - EID ExtendedID               → $ref: #/definitions/ExtendedID (named struct, R6-independent)
//
// Compare with `StoreOrder` above, where the equivalent fields
// dissolve to their primitive / open shapes because UUID and
// Anything carry no `swagger:model`.
//
// swagger:model order_modeled
type StoreOrderModeled struct {
	// the id for this order
	//
	// required: true
	// min: 1
	ID UUIDModeled `json:"id"`

	// the delivery option for this order
	DeliveryOption AnythingModeled `json:"deliveryOption"`

	// the extended ID (named struct, included as the R6-independent control)
	EID ExtendedID `json:"extended_id"`
}
