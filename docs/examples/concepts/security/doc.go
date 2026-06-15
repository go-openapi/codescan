// SPDX-License-Identifier: Apache-2.0

// snippet:meta

// Package security Reports API.
//
// The swagger:meta block declares the security schemes once and sets the
// document-wide default requirement.
//
//	Version: 1.0.0
//
//	SecurityDefinitions:
//	  api_key:
//	    type: apiKey
//	    in: header
//	    name: X-API-Key
//	  oauth2:
//	    type: oauth2
//	    flow: accessCode
//	    authorizationUrl: https://example.com/auth
//	    tokenUrl: https://example.com/token
//	    scopes:
//	      read: read reports
//	      write: write reports
//
//	Security:
//	  api_key:
//
// swagger:meta
package security

// endsnippet:meta
