// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package bug1096

/*
#include <stdlib.h>
*/
import "C"

// Order is used to foobar.
//
// swagger:response order
type Order struct {
	// in: body
	Body struct {
		// Name of the order
		Name string `json:"name"`
	}
}

// CreateOrder creates an order.
//
// swagger:route POST /orders orders createOrder
//
// Creates an order.
//
//	Responses:
//	  200: order
func CreateOrder() {
	_ = C.malloc(1)
}
