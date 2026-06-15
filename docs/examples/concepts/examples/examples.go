// SPDX-License-Identifier: Apache-2.0

// Package examples holds the annotated declarations used by the "Examples &
// defaults" tutorial. examples_test.go scans this package and writes the
// per-feature golden fragments the tutorial renders.
package examples

// snippet:example

// Greeting carries an example value for documentation.
//
// swagger:model
type Greeting struct {
	// Message is the greeting text.
	//
	// example: Hello, world!
	Message string `json:"message"`

	// Count is how many times to repeat it.
	//
	// example: 3
	Count int32 `json:"count"`
}

// endsnippet:example

// snippet:default

// Settings carries default values applied when a field is omitted.
//
// swagger:model
type Settings struct {
	// Port is the listen port.
	//
	// default: 8080
	Port int32 `json:"port"`

	// Mode is the run mode.
	//
	// default: auto
	Mode string `json:"mode"`

	// Verbose toggles verbose logging.
	//
	// default: false
	Verbose bool `json:"verbose"`
}

// endsnippet:default

// snippet:swaggerdefault

// DefaultPort is the fallback port used wherever Port is not supplied. The
// swagger:default annotation is a narrow value-only discovery hint.
//
// swagger:default
var DefaultPort = 8080 //nolint:gochecknoglobals // demo example

// endsnippet:swaggerdefault

// snippet:reffield

// Currency is a named (defined) string type, so it earns its own definition and
// a field typed Currency renders as a $ref. A $ref cannot carry sibling
// keywords, so an example or default on such a field rides the override arm of
// an allOf compound — the value still reaches the spec.
//
// swagger:model
type Currency string

// Price shows example + default on a defined-type field.
//
// swagger:model
type Price struct {
	// Unit is the ISO currency code.
	//
	// default: USD
	// example: EUR
	Unit Currency `json:"unit"`
}

// endsnippet:reffield

// snippet:responseexample

// NTPServers is a top-level array response carrying an example. The example
// lands on the response body schema rather than being dropped.
//
// swagger:response ntpServers
// example: ["10.0.0.1","10.0.0.2"]
type NTPServers []string

// swagger:route GET /ntp ntp listNTP
//
// responses:
//
//	200: ntpServers

// endsnippet:responseexample

// snippet:complexexample

// Profile carries structured (non-scalar) example values. A JSON object literal
// on a map field and a JSON array literal on a slice field are parsed into
// structured examples — a bare comma-separated list would instead be kept
// verbatim as a string.
//
// swagger:model
type Profile struct {
	// Labels is a set of key/value labels.
	//
	// example: {"env":"prod","tier":"gold"}
	Labels map[string]string `json:"labels"`

	// Roles is the list of assigned roles.
	//
	// example: ["admin","auditor"]
	Roles []string `json:"roles"`
}

// endsnippet:complexexample
