package loaders

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/calindra/nonodo/internal/reader/model"
	"github.com/jmoiron/sqlx"
	"github.com/vikstrous/dataloadgen"
)

type ctxKey string

const (
	loadersKey = ctxKey("dataloaders")
)

// userReader reads Users from a database
type userReader struct {
	db *sqlx.DB
}

// getReports implements a batch function that can retrieve many users by ID,
// for use in a dataloader
func (u *userReader) getReports(ctx context.Context, reportsKeys []string) ([]*model.Report, []error) {
	query := `
		SELECT output_index, payload, input_index 
		FROM convenience_reports 
		WHERE output_index IN (?)`
	query, args, err := sqlx.In(query, reportsKeys)
	if err != nil {
		slog.Error("query in", "error", err)
		return nil, []error{err}
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)
	stmt, err := u.db.PrepareContext(ctx, query)
	if err != nil {
		slog.Error("query", "error", err)
		return nil, []error{err}
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		slog.Error("query context execution",
			"error", err,
			"query", query,
			"args", args,
		)
		return nil, []error{err}
	}
	defer rows.Close()

	reports := make([]*model.Report, 0, len(reportsKeys))
	errs := make([]error, 0, len(reportsKeys))
	for rows.Next() {
		var report model.Report
		err := rows.Scan(&report.Index, &report.Payload, &report.InputIndex)
		reports = append(reports, &report)
		errs = append(errs, err)
	}
	slog.Debug("get reports", "reports", reportsKeys)
	return reports, errs
}

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	ReportLoader *dataloadgen.Loader[string, *model.Report]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(conn *sqlx.DB) *Loaders {
	// define the data loader
	ur := &userReader{db: conn}
	return &Loaders{
		ReportLoader: dataloadgen.NewLoader(
			ur.getReports,
			dataloadgen.WithWait(time.Millisecond),
		),
	}
}

// Middleware injects data loaders into the context
func Middleware(conn *sqlx.DB, next http.Handler) http.Handler {
	// return a middleware that injects the loader to the request context
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loader := NewLoaders(conn)
		r = r.WithContext(context.WithValue(r.Context(), loadersKey, loader))
		next.ServeHTTP(w, r)
	})
}

// For returns the dataloader for a given context
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

// GetReports returns single reports by reportsKey efficiently
func GetReports(ctx context.Context, reportsKey string) (*model.Report, error) {
	loaders := For(ctx)
	return loaders.ReportLoader.Load(ctx, reportsKey)
}

// GetMayReports returns many reports by reportsKeys efficiently
func GetMayReports(ctx context.Context, reportsKeys []string) ([]*model.Report, error) {
	loaders := For(ctx)
	return loaders.ReportLoader.LoadAll(ctx, reportsKeys)
}
