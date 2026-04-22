// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package routebody

import "errors"

// ErrParser is the sentinel error for failures originating in the
// swagger:route body parsers (parameters / responses / extensions).
var ErrParser = errors.New("codescan:parsers/routebody")
