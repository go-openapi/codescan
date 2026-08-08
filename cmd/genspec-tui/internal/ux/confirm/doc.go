// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package confirm is the yes or no modal: a question that must be answered before something irreversible happens.
//
// It follows the same contract as the other overlays - a concrete type the root model owns and drives, never a
// tea.Model. The overlay asks the question and records the answer; what the answer MEANS is the model's to decide, so
// nothing here knows what is being confirmed.
package confirm
