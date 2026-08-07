// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package testutils holds helpers shared by the tests of several ux packages.
//
// Test-only, and imported from _test.go files alone. [StripANSI] is the one that matters: the panels are tested with a
// colour profile forced on, so nearly every assertion needs the text without the styling.
package testutils
