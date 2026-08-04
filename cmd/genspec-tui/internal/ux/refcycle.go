// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
)

// RefCycle is the find-references walk (F3 / shift+F3): the definition whose uses are being visited, its reference
// sites in rendered-line order, and which one the cursor is parked on.
//
// It is valid only for the render it was computed against — every line number in it is a line of THAT render — so a
// rescan or a format toggle drops it.
type RefCycle struct {
	Anchor string          // the definition pointer whose uses are being cycled
	Sites  []index.RefSite // its reference sites, ordered by rendered line
	Cursor int             // which site we are parked on
	Status string          // persistent status line while a cycle is active
}

// ParkedOn reports whether a cycle is under way and line is still the site it last moved to.
//
// This is what makes "F3 repeatedly" walk one definition's uses: move the cursor off the site and the next step
// re-anchors on the node you are now on, rather than chasing the definition of whatever it last landed on.
func (c *RefCycle) ParkedOn(line int) bool {
	return c.Anchor != "" && c.Cursor < len(c.Sites) && c.Sites[c.Cursor].Line == line
}

// Start anchors a fresh cycle, entering at the first site for a forward step and the last for a backward one.
func (c *RefCycle) Start(anchor string, sites []index.RefSite, dir int) {
	c.Anchor, c.Sites, c.Cursor = anchor, sites, 0
	if dir < 0 {
		c.Cursor = len(sites) - 1
	}
}

// Step moves to the next (dir +1) or previous (-1) site, wrapping at both ends.
func (c *RefCycle) Step(dir int) {
	c.Cursor = (c.Cursor + dir + len(c.Sites)) % len(c.Sites)
}

// Site is the reference site the cycle is currently parked on.
func (c *RefCycle) Site() index.RefSite { return c.Sites[c.Cursor] }

// Reset drops the cycle.
//
// Called whenever the render it was computed against is replaced (rescan, format toggle) or the user moves on.
func (c *RefCycle) Reset() { *c = RefCycle{} }

// Describe renders the status line: which site of how many, of what, pointing where.
func (c *RefCycle) Describe() string {
	return fmt.Sprintf("ref %d/%d of %s → %s", c.Cursor+1, len(c.Sites), c.Anchor, c.Site().Pointer)
}
