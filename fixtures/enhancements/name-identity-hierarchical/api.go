// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package hierarchical is the name-identity witness for the hierarchical
// fail-safe: two packages with LONG leaf names each declare
// `swagger:model Config`. The leaf "Config" collides, and the flat
// minimal concat (RecommendationengineConfig / NotificationserviceConfig)
// scores over the readability budget. With EmitHierarchicalNames set, the
// reduce stage emits nested container definitions
// (#/definitions/recommendationengine/Config, …) instead; by default it
// keeps the always-correct flat concat.
package hierarchical

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-hierarchical/notificationservice"
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-hierarchical/recommendationengine"
)

// swagger:model Settings
type Settings struct {
	Rec   recommendationengine.Config `json:"rec"`
	Notif notificationservice.Config  `json:"notif"`
}
