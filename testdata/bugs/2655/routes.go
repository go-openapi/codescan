// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2655

// listPets lists the pets.
//
// The route declares one tag on the swagger:route header line (`pets`)
// and two more via a body `Tags:` keyword block (`pets`, `store`). The
// builder unions them, deduping `pets`, so op.Tags == [pets, store]
// (go-swagger#2655).
//
// swagger:route GET /pets pets listPets
//
//	Tags:
//	  - pets
//	  - store
//
//	Responses:
//	  200: description: OK
func listPets() {}

// getPet fetches one pet.
//
// A swagger:operation wholesale-unmarshals its YAML body, so a `tags:`
// list there already lands on op.Tags as a string list — pinned here
// alongside the route-keyword path (go-swagger#2655).
//
// swagger:operation GET /pets/{id} getPet
//
// ---
// tags:
//   - pets
//   - store
// responses:
//   "200":
//     description: OK
func getPet() {}
