import type { SourceFile } from './types';

// What the playground opens with, and what the gallery offers.
//
// Each is a whole module - go.mod included - because that is what the scanner takes, and because a
// visitor who wants to try their own code should be looking at something shaped like their own code.
// Each was written as a real module on disk, run through the artifact, and its output read before
// being pasted in - an example that does not produce what it claims to teach is worse than none.
//
// A shortcode will be able to replace this wholesale, which is why the files are built fresh on each
// call rather than shared.

export type Example = {
  id: string;
  title: string;
  /** One line: what this one shows that the others do not. */
  blurb: string;
  files: () => SourceFile[];
};

export const examples: Example[] = [
  {
    id: "models",
    title: "Models",
    blurb: "A struct becomes a definition: validations, an enum, an example, and a $ref to another model.",
    files: () => [
      { path: "go.mod", text: `module example.com/models

go 1.25.0
` },
      { path: "api/pet.go", text: `// Package api describes the store's data.
package api

// Pet is an animal in the store.
//
// swagger:model pet
type Pet struct {
	// The pet's identifier
	//
	// required: true
	// minimum: 1
	ID int64 \`json:"id"\`

	// The pet's name
	//
	// required: true
	// max length: 50
	// example: Fido
	Name string \`json:"name"\`

	// What sort of animal it is
	//
	// enum: cat,dog,bird
	Kind string \`json:"kind"\`

	// Free-form labels
	//
	// unique: true
	Tags []string \`json:"tags,omitempty"\`

	// Where it lives, if anywhere
	Address *Address \`json:"address,omitempty"\`
}

// Address is a place a pet may live.
//
// swagger:model address
type Address struct {
	// pattern: ^[A-Z]{2}$
	Country string \`json:"country"\`
	City    string \`json:"city"\`
}
` },
    ],
  },
  {
    id: "routes",
    title: "Routes",
    blurb: "swagger:route with its parameters and responses \u2014 the shape most APIs are written in.",
    files: () => [
      { path: "go.mod", text: `module example.com/routes

go 1.25.0
` },
      { path: "api/handlers.go", text: `// Package api describes the store's endpoints.
package api

// swagger:route GET /pets pets listPets
//
// Lists the pets in the store.
//
// Responses:
//   200: petList
//   default: errorBody

// swagger:route POST /pets pets addPet
//
// Adds a pet to the store.
//
// Responses:
//   201: petBody
//   422: errorBody

// ListPetsParams is the query for listing pets.
//
// swagger:parameters listPets
type ListPetsParams struct {
	// How many to return at most
	//
	// in: query
	// minimum: 1
	// maximum: 100
	// default: 20
	Limit int32 \`json:"limit"\`

	// Only pets of this kind
	//
	// in: query
	Kind string \`json:"kind"\`
}

// PetList is the list of pets.
//
// swagger:response petList
type PetList struct {
	// in: body
	Body []Pet
}

// PetBody is one pet.
//
// swagger:response petBody
type PetBody struct {
	// in: body
	Body Pet
}

// ErrorBody explains what went wrong.
//
// swagger:response errorBody
type ErrorBody struct {
	// in: body
	Body struct {
		Message string \`json:"message"\`
		Code    int32  \`json:"code"\`
	}
}

// Pet is an animal in the store.
//
// swagger:model pet
type Pet struct {
	// required: true
	ID int64 \`json:"id"\`
	// required: true
	Name string \`json:"name"\`
}
` },
    ],
  },
  {
    id: "operation",
    title: "Operation",
    blurb: "swagger:operation, where the body is YAML and you write the OpenAPI directly.",
    files: () => [
      { path: "go.mod", text: `module example.com/operation

go 1.25.0
` },
      { path: "api/handlers.go", text: `// Package api describes one endpoint in full.
package api

// swagger:operation GET /pets/{id} pets getPet
//
// ---
// summary: Fetches one pet by its identifier.
// description: |
//   Returns the pet, or 404 when the store has never heard of it.
//
//   The identifier is the one returned when the pet was added.
// parameters:
//   - name: id
//     in: path
//     description: the pet's identifier
//     type: integer
//     format: int64
//     required: true
//     minimum: 1
// responses:
//   "200":
//     description: the pet
//     schema:
//       $ref: "#/definitions/pet"
//   "404":
//     description: no such pet

// Pet is an animal in the store.
//
// swagger:model pet
type Pet struct {
	// required: true
	ID int64 \`json:"id"\`
	// required: true
	Name string \`json:"name"\`
}
` },
    ],
  },
  {
    id: "enums",
    title: "Enums",
    blurb: "A Go constant set becomes an enum, typed from the declaration \u2014 including an iota block.",
    files: () => [
      { path: "go.mod", text: `module example.com/enums

go 1.25.0
` },
      { path: "api/kind.go", text: `// Package api shows how a Go constant set becomes an enum.
package api

// Kind is what sort of animal a pet is.
//
// swagger:enum Kind
type Kind string

const (
	// KindCat is a cat.
	KindCat Kind = "cat"
	// KindDog is a dog.
	KindDog Kind = "dog"
	// KindBird is a bird.
	KindBird Kind = "bird"
)

// Size is how big the animal is, counted rather than named.
//
// swagger:enum Size
type Size int8

const (
	Small Size = iota + 1
	Medium
	Large
)

// Pet is an animal in the store.
//
// swagger:model pet
type Pet struct {
	// required: true
	Kind Kind \`json:"kind"\`
	Size Size \`json:"size"\`
}
` },
    ],
  },
  {
    id: "polymorphism",
    title: "Polymorphism",
    blurb: "A discriminated base and its subtypes, composed with swagger:allOf.",
    files: () => [
      { path: "go.mod", text: `module example.com/polymorphism

go 1.25.0
` },
      { path: "api/pet.go", text: `// Package api shows a discriminated family.
package api

// Pet is the base every animal shares.
//
// swagger:model pet
type Pet struct {
	// required: true
	ID int64 \`json:"id"\`

	// What sort of animal this is. The discriminator: its value names the subtype.
	//
	// required: true
	// discriminator: true
	Kind string \`json:"kind"\`
}

// Cat is a pet that sleeps a great deal.
//
// swagger:model cat
type Cat struct {
	// swagger:allOf
	Pet

	// How aloof, from nought to ten
	//
	// maximum: 10
	Aloofness int32 \`json:"aloofness"\`
}

// Dog is a pet that does not.
//
// swagger:model dog
type Dog struct {
	// swagger:allOf
	Pet

	// Whether it fetches
	Fetches bool \`json:"fetches"\`
}
` },
    ],
  },
];

/** What opens on a cold visit. Models first: it is the smallest complete thing codescan does. */
export const firstExample = 'models';

export function exampleById(id: string): Example {
  return examples.find((e) => e.id === id) ?? examples[0];
}

// sampleFiles is what the store starts from.
export function sampleFiles(): SourceFile[] {
  return exampleById(firstExample).files();
}
