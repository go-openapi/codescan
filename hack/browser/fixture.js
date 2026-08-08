// The module the probe scans. Kept in one place so the worker and any later app agree on it.
export const goMod = `module example.com/demo

go 1.25.0
`;

export const petSrc = `package models

// Pet is an animal in the store.
//
// swagger:model pet
type Pet struct {
	// The pet's identifier
	// required: true
	ID int64 \`json:"id"\`

	// The pet's name
	// max length: 50
	Name string \`json:"name"\`
}
`;
