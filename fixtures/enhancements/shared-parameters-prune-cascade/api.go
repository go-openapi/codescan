// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package shared_parameters_prune_cascade witnesses the second step of the
// shared-parameters prune extension (C4 / P7, go-swagger#2632): pruning an
// unreferenced shared response also prunes the definitions reachable ONLY
// through it (the cascade), while definitions still reachable via a surviving
// shared object are kept.
//
// The shared-object prune runs BEFORE the definition reachability walk, so a
// model whose last keeper was a now-pruned shared response loses its root and
// is pruned in turn — but a model shared with a surviving response stays.
//
// Layout (scanned under ScanModels, toggling PruneUnusedModels):
//
//	UsedResponse  (swagger:response *, referenced by listC) → body Survivor
//	UnusedResponse(swagger:response *, referenced by nobody) → body Orphan
//	Survivor  → field Shared           (kept: UsedResponse survives)
//	Orphan    → field Shared           (pruned: only UnusedResponse reached it)
//	Shared                              (kept: still reachable via Survivor)
//
// Expected with PruneUnusedModels: responses {UsedResponse}; definitions
// {Survivor, Shared}; UnusedResponse + Orphan pruned (two scan.pruned-unused
// Hints); Shared NOT pruned (the prune is reachability-correct, not a naive
// "drop everything the pruned response touched").
//
// See .claude/plans/features/shared-parameters-fixtures.md (§6b, step 2).
package shared_parameters_prune_cascade

// Survivor is reached from UsedResponse's body and survives the prune.
//
// swagger:model
type Survivor struct {
	Name   string `json:"name"`
	Shared Shared `json:"shared"`
}

// Orphan is reached ONLY from UnusedResponse's body. When UnusedResponse is
// pruned (no operation references it), Orphan loses its only keeper and is
// pruned in turn — the cascade.
//
// swagger:model
type Orphan struct {
	Detail string `json:"detail"`
	Shared Shared `json:"shared"`
}

// Shared is reached from BOTH Survivor (kept) and Orphan (pruned). It must
// survive, because the surviving Survivor still references it — the cascade
// prunes only what nothing reachable keeps alive.
//
// swagger:model
type Shared struct {
	Code int64 `json:"code"`
}

// UsedResponse is referenced by listC's Responses block → survives, keeping
// Survivor (and transitively Shared).
//
// swagger:response *
type UsedResponse struct {
	// in: body
	Body Survivor `json:"body"`
}

// UnusedResponse is referenced by no operation → pruned, cascading to Orphan.
//
// swagger:response *
type UnusedResponse struct {
	// in: body
	Body Orphan `json:"body"`
}

// ListC references only the used shared response; UnusedResponse is left
// dangling for the prune pass.
//
// swagger:route GET /c cascade listC
// Responses:
//
//	default: UsedResponse
func ListC() {}
