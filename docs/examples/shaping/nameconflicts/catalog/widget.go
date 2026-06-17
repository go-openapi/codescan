// SPDX-License-Identifier: Apache-2.0

// Package catalog provides a model referenced by leaf name from a type-name
// keyword in another package (see Bag in the root package). Because "Widget" is
// unique across the scanned packages, the bare leaf resolves to this model.
package catalog

// snippet:catalog

// Widget is referenced cross-package by its bare leaf name "Widget".
//
// swagger:model Widget
type Widget struct {
	SKU string `json:"sku"`
}

// endsnippet:catalog
