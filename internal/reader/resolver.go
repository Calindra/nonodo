package reader

import (
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/reader/model"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	model              *model.ModelWrapper
	convenienceService *convenience.ConvenienceService
}
