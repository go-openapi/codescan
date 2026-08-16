// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1613

// ServiceStatusResponse is a response with a plain string body. The reporter
// saw the response emitted with no schema (description only).
//
// swagger:response serviceStatusResponse
type ServiceStatusResponse struct {
	// Status
	//
	// in: body
	Status string
}

// GetServiceStatus reports service status.
//
// swagger:route GET /status status GetServiceStatus
//
// Service status.
//
//	Responses:
//	  200: serviceStatusResponse
func GetServiceStatus() {}
