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
var DefaultPort = 8080

// endsnippet:swaggerdefault
