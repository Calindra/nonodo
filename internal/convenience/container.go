package convenience

import (
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/jmoiron/sqlx"
)

// what is the best IoC for go?

type Container struct {
	db                 *sqlx.DB
	outputDecoder      *OutputDecoder
	convenienceService *ConvenienceService
	repository         *repository.VoucherRepository
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
	c.outputDecoder = NewOutputDecoder(*c.GetConvenienceService())
	return c.outputDecoder
}

func (c *Container) GetRepository() *repository.VoucherRepository {
	if c.repository != nil {
		return c.repository
	}
	c.repository = &repository.VoucherRepository{
		Db: *c.db,
	}
	err := c.repository.CreateTables()
	if err != nil {
		panic(err)
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