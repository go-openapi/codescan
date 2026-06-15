// SPDX-License-Identifier: Apache-2.0

// snippet:meta

// Package meta Pet Store.
//
// A small API that demonstrates the document-level swagger:meta block: the
// package doc comment carries the spec's top-level metadata.
//
//	Schemes: https
//	Host: api.example.com
//	BasePath: /v1
//	Version: 1.2.0
//	License: Apache 2.0 https://www.apache.org/licenses/LICENSE-2.0.html
//	Contact: API Team <api@example.com> https://example.com/support
//
//	Consumes:
//	  - application/json
//
//	Produces:
//	  - application/json
//
//	ExternalDocs:
//	  description: Full API guide
//	  url: https://example.com/docs
//
//	Tags:
//	- name: pets
//	  description: Everything about your Pets
//	  externalDocs:
//	    description: Find out more
//	    url: https://example.com/docs/pets
//	- name: store
//	  description: Access to Petstore orders
//	  x-display-name: Store
//
// swagger:meta
package meta

// endsnippet:meta
