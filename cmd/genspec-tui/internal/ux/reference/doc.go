// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package reference is the annotation-reference popup: what the swagger: directive on this line means, and what may be
// written under it, without leaving the file to go and look it up.
//
// It follows the same contract as the other overlays - a concrete type the root model owns and drives, never a
// tea.Model. Unlike them it records nothing: there is no answer to collect, only something to read and dismiss.
package reference
