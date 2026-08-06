// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package packages

// AnnotationChunk exposes the marker scan's read size so a test can place a marker exactly on the boundary it reads
// at. Pinning the boundary in the test rather than a literal means the test follows the constant if it is retuned,
// instead of quietly ceasing to straddle anything.
const AnnotationChunk = annotationChunk
