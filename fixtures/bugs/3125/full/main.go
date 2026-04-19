//go:build testintegration

// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"net"
	"net/http"
	"swagger/api"
)

func main() {
	// Route => handler
	http.HandleFunc("POST /foobar", api.FooBarHandler)

	// Start server
	listener, err := net.Listen("tcp", ":1323")
	if err != nil {
		log.Fatal(err)
	}

	err = http.Serve(listener, nil)
	if err != nil {
		log.Fatal(err)
	}
}
