// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package help is the keymap overlay: a scrollable modal listing every binding, grouped by the context it applies in.
//
// It follows the same contract as the panels - a concrete type the root model owns and drives, never a tea.Model. An
// overlay that took the root model as a parameter would only be handing it straight back, and deciding app policy
// (quitting) on its behalf.
package help
