// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug1121 probes whether tag descriptions (the OAI root-level tags
// section, with description / externalDocs per tag) can be declared from code.
//
//	Schemes: https
//	Host: localhost
//	Version: 1.0.0
//
//	Tags:
//	- name: users
//	  description: Operations about users
//
// swagger:meta
package bug1121
