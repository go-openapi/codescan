// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2655 reproduces go-swagger issue #2655 (tag metadata ignored): a
// tag list in swagger:meta must populate the top-level tags section (with
// per-tag descriptions, externalDocs and vendor extensions), not be swallowed
// into info.description.
//
//	Version: 1.0.0
//	Tags:
//	- name: pet
//	  description: Everything about your Pets
//	  externalDocs:
//	    description: Find out more
//	    url: http://swagger.io
//	- name: store
//	  description: Access to Petstore orders
//	  x-display-name: Store
//
// swagger:meta
package bug2655
