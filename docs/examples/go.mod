// Module github.com/go-openapi/codescan/docs/examples hosts runnable code
// samples referenced from the documentation site. It is intentionally kept
// separate from the root module so example dependencies do not leak into
// codescan consumers.
module github.com/go-openapi/codescan/docs/examples

go 1.25.0

require (
	github.com/go-openapi/codescan v0.0.0
	github.com/go-openapi/spec v0.22.9
	github.com/go-openapi/testify/v2 v2.6.0
)

require (
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/go-openapi/jsonreference v1.0.0 // indirect
	github.com/go-openapi/swag/conv v0.28.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.28.0 // indirect
	github.com/go-openapi/swag/loading v0.28.0 // indirect
	github.com/go-openapi/swag/mangling v0.28.0 // indirect
	github.com/go-openapi/swag/pools v0.28.0 // indirect
	github.com/go-openapi/swag/stringutils v0.28.0 // indirect
	github.com/go-openapi/swag/typeutils v0.28.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

replace github.com/go-openapi/codescan => ../..
