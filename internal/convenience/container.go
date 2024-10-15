package convenience

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/calindra/nonodo/internal/convenience/decoder"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/calindra/nonodo/internal/convenience/synchronizer"
	"github.com/calindra/nonodo/internal/graphile"
	"github.com/jmoiron/sqlx"
)

// what is the best DI/IoC framework for go?

type Container struct {
	db                    *sqlx.DB
	outputDecoder         *decoder.OutputDecoder
	convenienceService    *services.ConvenienceService
	outputRepository      *repository.OutputRepository
	repository            *repository.VoucherRepository
	syncRepository        *repository.SynchronizerRepository
	graphQLSynchronizer   *synchronizer.Synchronizer
	voucherFetcher        *synchronizer.VoucherFetcher
	noticeRepository      *repository.NoticeRepository
	graphileFetcher       *synchronizer.GraphileFetcher
	graphileSynchronizer  *synchronizer.GraphileSynchronizer
	graphileClient        graphile.GraphileClient
	inputRepository       *repository.InputRepository
	reportRepository      *repository.ReportRepository
	AutoCount             bool
	rawInputRefRepository *repository.RawInputRefRepository
}

func NewContainer(db sqlx.DB, autoCount bool) *Container {
	return &Container{
		db:        &db,
		AutoCount: autoCount,
	}
}

func (c *Container) GetOutputDecoder() *decoder.OutputDecoder {
	if c.outputDecoder != nil {
		return c.outputDecoder
	}
	c.outputDecoder = decoder.NewOutputDecoder(*c.GetConvenienceService())
	return c.outputDecoder
}

func (c *Container) GetOutputRepository() *repository.OutputRepository {
	if c.outputRepository != nil {
		return c.outputRepository
	}
	c.outputRepository = &repository.OutputRepository{
		Db: *c.db,
	}
	// err := c.outputRepository.CreateTables()
	// if err != nil {
	// 	panic(err)
	// }
	return c.outputRepository
}

func (c *Container) GetVoucherRepository() *repository.VoucherRepository {
	if c.repository != nil {
		return c.repository
	}
	c.repository = &repository.VoucherRepository{
		Db:               *c.db,
		OutputRepository: *c.GetOutputRepository(),
		AutoCount:        c.AutoCount,
	}
	err := c.repository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.repository
}

func (c *Container) GetSyncRepository() *repository.SynchronizerRepository {
	if c.syncRepository != nil {
		return c.syncRepository
	}
	c.syncRepository = &repository.SynchronizerRepository{
		Db: *c.db,
	}
	err := c.syncRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.syncRepository
}

func (c *Container) GetNoticeRepository() *repository.NoticeRepository {
	if c.noticeRepository != nil {
		return c.noticeRepository
	}
	c.noticeRepository = &repository.NoticeRepository{
		Db:               *c.db,
		OutputRepository: *c.GetOutputRepository(),
		AutoCount:        c.AutoCount,
	}
	err := c.noticeRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.noticeRepository
}

func (c *Container) GetRawInputRepository() *repository.RawInputRefRepository {
	if c.rawInputRefRepository != nil {
		return c.rawInputRefRepository
	}
	c.rawInputRefRepository = &repository.RawInputRefRepository{
		Db: *c.db,
	}
	err := c.rawInputRefRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.rawInputRefRepository
}

func (c *Container) GetInputRepository() *repository.InputRepository {
	if c.inputRepository != nil {
		return c.inputRepository
	}
	c.inputRepository = &repository.InputRepository{
		Db: *c.db,
	}
	err := c.inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.inputRepository
}

func (c *Container) GetReportRepository() *repository.ReportRepository {
	if c.reportRepository != nil {
		return c.reportRepository
	}
	c.reportRepository = &repository.ReportRepository{
		Db:        c.db,
		AutoCount: c.AutoCount,
	}
	err := c.reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.reportRepository
}

func (c *Container) GetConvenienceService() *services.ConvenienceService {
	if c.convenienceService != nil {
		return c.convenienceService
	}
	c.convenienceService = services.NewConvenienceService(
		c.GetVoucherRepository(),
		c.GetNoticeRepository(),
		c.GetInputRepository(),
		c.GetReportRepository(),
	)
	return c.convenienceService
}

func (c *Container) GetGraphQLSynchronizer() *synchronizer.Synchronizer {
	if c.graphQLSynchronizer != nil {
		return c.graphQLSynchronizer
	}
	c.graphQLSynchronizer = synchronizer.NewSynchronizer(
		c.GetOutputDecoder(),
		c.GetVoucherFetcher(),
		c.GetSyncRepository(),
	)
	return c.graphQLSynchronizer
}

func (c *Container) GetVoucherFetcher() *synchronizer.VoucherFetcher {
	if c.voucherFetcher != nil {
		return c.voucherFetcher
	}
	c.voucherFetcher = synchronizer.NewVoucherFetcher()
	return c.voucherFetcher
}

func (c *Container) GetGraphileSynchronizer(graphileUrl url.URL, loadTestMode bool) *synchronizer.GraphileSynchronizer {
	if c.graphileSynchronizer != nil {
		return c.graphileSynchronizer
	}

	graphileClient := c.GetGraphileClient(graphileUrl, loadTestMode)
	c.graphileSynchronizer = synchronizer.NewGraphileSynchronizer(
		c.GetOutputDecoder(),
		c.GetSyncRepository(),
		c.GetGraphileFetcher(graphileClient),
	)
	return c.graphileSynchronizer
}

func (c *Container) GetGraphileFetcher(graphileClient graphile.GraphileClient) *synchronizer.GraphileFetcher {
	if c.graphileFetcher != nil {
		return c.graphileFetcher
	}
	c.graphileFetcher = synchronizer.NewGraphileFetcher(graphileClient)
	return c.graphileFetcher
}

func (c *Container) GetGraphileClient(graphileUrl url.URL, loadTestMode bool) graphile.GraphileClient {

	if c.graphileClient != nil {
		return c.graphileClient
	}

	if loadTestMode {
		const serviceName = "http://postgraphile"
		if graphileUrl.Port() != "" {
			graphileUrl.Host = fmt.Sprintf("%s:%s", serviceName, graphileUrl.Port())
		} else {
			graphileUrl.Host = serviceName
		}
	}
	slog.Debug("GraphileClient",
		"graphileUrl", graphileUrl,
	)
	c.graphileClient = &graphile.GraphileClientImpl{
		GraphileUrl: graphileUrl,
	}
	return c.graphileClient
}
