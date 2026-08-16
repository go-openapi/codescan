// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1416

// Cluster is the body model.
//
// swagger:model cluster
type Cluster struct {
	Name string `json:"name"`
}

// CreateCluster mirrors the reporter's swagger:operation with an inline body
// parameter whose schema is a $ref. The reporter saw a spurious top-level $ref
// emitted alongside the schema (malformed param). Check the param is clean.
//
// swagger:operation POST /api/v1/clusters createCluster
//
// Creates a cluster.
//
// ---
// produces:
// - text/plain
// consumes:
// - application/json
// parameters:
// - name: cluster
//   in: body
//   description: the cluster specification to create
//   required: true
//   schema:
//     "$ref": "#/definitions/cluster"
// responses:
//   '202':
//     description: Accepted
func CreateCluster() {}
