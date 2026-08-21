module github.com/go-openapi/codescan/cmd/genspec

// The library is required at the version that will carry internal/cliopts, which this command reads
// its flag surface from. Until codescan releases it, this module builds through the workspace at the
// repository root - which is how CI tests it - and `go install .../cmd/genspec@latest` will resolve
// once that release is out.
go 1.25.0

require (
	github.com/SladkyCitron/slogcolor v1.9.0
	github.com/charmbracelet/x/term v0.2.2
	github.com/go-openapi/codescan v0.36.4
	github.com/go-openapi/loads v0.25.1
	github.com/go-openapi/spec v0.22.9
	github.com/go-openapi/strfmt v0.27.0
	github.com/go-openapi/swag/conv v0.28.0
	github.com/go-openapi/testify/v2 v2.6.1
	github.com/go-openapi/validate v0.26.3
	github.com/knadh/koanf/providers/rawbytes v1.0.1
	github.com/knadh/koanf/v2 v2.3.6
	go.yaml.in/yaml/v3 v3.0.5
)

replace github.com/go-openapi/codescan => ../..

require (
	github.com/fatih/color v1.19.0 // indirect
	github.com/go-openapi/analysis v0.26.0 // indirect
	github.com/go-openapi/errors v0.22.8 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/go-openapi/jsonreference v1.0.0 // indirect
	github.com/go-openapi/swag/fileutils v0.28.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.28.0 // indirect
	github.com/go-openapi/swag/loading v0.28.0 // indirect
	github.com/go-openapi/swag/mangling v0.28.0 // indirect
	github.com/go-openapi/swag/pools v0.28.0 // indirect
	github.com/go-openapi/swag/stringutils v0.28.0 // indirect
	github.com/go-openapi/swag/typeutils v0.28.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.28.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/knadh/koanf/maps v0.1.3 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
)
