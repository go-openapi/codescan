// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1958

// swagger:operation GET /aws aws getAws
//
// AWS-integrated operation.
//
// ---
// responses:
//   '200':
//     description: ok
// x-amazon-apigateway-integration:
//   httpMethod: GET
//   passthroughBehavior: when_no_match
//   responses:
//     default:
//       statusCode: "200"
//   type: http_proxy
//   uri: https://proxy-url.com
func GetAws() {}
