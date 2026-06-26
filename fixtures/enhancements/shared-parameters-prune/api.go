// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package shared_parameters_prune (Fixture 6) witnesses the prune
// extension (C4) of the shared-parameters feature (go-swagger#2632):
// shared parameters/responses referenced by no operation are pruned
// under PruneUnusedModels, and retained without it.
//
// The integration test scans this package twice — with ScanModels only
// (all four shared objects present) and with ScanModels +
// PruneUnusedModels (the unused pair pruned) — mirroring
// coverage_prune_unused_test.go.
//
// See .claude/plans/features/shared-parameters-fixtures.md (§6b).
package shared_parameters_prune

// UsedHeader registers #/parameters/X-Used and is referenced by listP →
// it survives pruning.
//
// swagger:parameters *
type UsedHeader struct {
	// in: header
	Used string `json:"X-Used"`
}

// UnusedHeader registers #/parameters/X-Unused and is referenced by no
// operation → pruned under PruneUnusedModels.
//
// swagger:parameters *
type UnusedHeader struct {
	// in: header
	Unused string `json:"X-Unused"`
}

// UsedResponse is referenced by listP's Responses block → survives.
//
// swagger:response *
type UsedResponse struct {
	// in: body
	Body struct {
		// OK message.
		Message string `json:"message"`
	} `json:"body"`
}

// UnusedResponse is referenced by no operation → pruned under
// PruneUnusedModels.
//
// swagger:response *
type UnusedResponse struct {
	// in: body
	Body struct {
		// unused detail.
		Detail string `json:"detail"`
	} `json:"body"`
}

// ListP references the used shared parameter and the used shared
// response; the unused pair is left dangling for the prune pass.
//
// swagger:route GET /p prune listP
// swagger:parameters listP X-Used
// Responses:
//
//	default: UsedResponse
func ListP() {}
