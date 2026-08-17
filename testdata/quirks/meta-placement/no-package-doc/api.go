// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package metanopackagedoc

// The meta block, in a file whose package clause carries no doc comment of its own.
//
// The blank line above the package clause is what makes the file's doc nil, and recording the
// info node's cross-reference anchor used to dereference it.
//
// Version: 4.5.6
// Host: nodoc.example.com
//
// swagger:meta

// Widget is a widget.
//
// swagger:model Widget
type Widget struct {
	// ID of the widget.
	ID string `json:"id"`
}
