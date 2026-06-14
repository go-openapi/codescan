// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1609

// request carries a path param with a vendor extension declared via an
// Extensions: block. Parameters ALREADY honor Extensions: — this is the green
// guard rail.
//
// swagger:parameters UserDestroy
type request struct {
	// in: path
	//
	// Extensions:
	//   x-example: USERID
	UserID string `json:"user_id"`
}

// StatusResponse carries a header with an Extensions: block. Response headers do
// NOT yet honor Extensions: (the #1609 gap, reframed): the x-units extension is
// dropped from the emitted header.
//
// swagger:response statusResponse
type StatusResponse struct {
	// requests allowed per window
	//
	// Extensions:
	//   x-units: requests-per-minute
	XRateLimit int `json:"X-Rate-Limit"`
}

// DeleteUser uses the UserDestroy parameters.
//
// swagger:route DELETE /users/{user_id} users UserDestroy
//
// Delete a user.
//
//	Responses:
//	  200: statusResponse
func DeleteUser() {}
