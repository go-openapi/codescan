// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/go-openapi/codescan/fixtures/bugs/2907/data"

// swagger:route GET /movies movies listMovies
//
// responses:
//   200: moviesResponse

// A list of movies
//
// swagger:response moviesResponse
type MoviesResponse struct {
	// in: body
	Body []data.Movie
}
