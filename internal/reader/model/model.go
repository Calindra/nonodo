// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This module is a wrapper for the nonodo model that converts the internal types to
// GraphQL-compatible types.
package model

import (
	"github.com/cartesi/rollups-graphql/pkg/model"
)

// Nonodo model wrapper that convert types to GraphQL types.
type ModelWrapper struct {
	model *model.NonodoModel
}

func NewModelWrapper(model *model.NonodoModel) *ModelWrapper {
	return &ModelWrapper{model}
}
