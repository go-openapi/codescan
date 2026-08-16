// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1852

// swagger:route DELETE /orders/{id} orders deleteOrder
//
// Delete an order.
//
// Responses:
//
//	204: noContentResponse
func DeleteOrder() {}

// noContentResponse has NO description comment — the #1852 trigger. The legacy
// engine omitted the OAS2-required `description` key entirely (editor.swagger.io
// reported "missingProperty: description"); codescan now always emits it.
//
// swagger:response noContentResponse
type noContentResponse struct{}
