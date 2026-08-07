// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package validation runs the produced spec through go-openapi/validate and normalises what comes back into findings
// the TUI can list and navigate to.
//
// This asks a different question from the scanner's own diagnostics. Those say whether the ANNOTATIONS were understood;
// these say whether the DOCUMENT they produced is a legal Swagger 2.0 spec. A scan can be perfectly clean and still
// emit something a consumer will reject, which is the gap this closes.
package validation
