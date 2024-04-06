package convenience

import (
	"github.com/jmoiron/sqlx"
)

type Container struct {
	db                 *sqlx.DB
	outputDecoder      *OutputDecoder
	convenienceService *ConvenienceService
	repository         *ConvenienceRepositoryImpl
}

func NewContainer(db sqlx.DB) *Container {
	return &Container{
		db: &db,
	}
}

func (c *Container) GetOutputDecoder() *OutputDecoder {
	if c.outputDecoder != nil {
		return c.outputDecoder
	}
	c.outputDecoder = NewOutputDecoder(c.db)
	return c.outputDecoder
}

func (c *Container) GetRepository() *ConvenienceRepositoryImpl {
	if c.repository != nil {
		return c.repository
	}
	c.repository = &ConvenienceRepositoryImpl{
		db: *c.db,
	}
	return c.repository
}

func (c *Container) GetConvenienceService() *ConvenienceService {
	if c.convenienceService != nil {
		return c.convenienceService
	}
	c.convenienceService = &ConvenienceService{
		repository: c.GetRepository(),
	}
	return c.convenienceService
}
