package repository

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type OutputRepository struct {
	Db sqlx.DB
}

func (c *OutputRepository) CountAllOutputs(
	ctx context.Context,
) (uint64, error) {
	vouchers, err := c.CountAllVouchers(ctx)
	if err != nil {
		slog.Error("query error")
		return 0, err
	}
	notices, err := c.CountAllNotices(ctx)
	if err != nil {
		slog.Error("query error")
		return 0, err
	}
	return vouchers + notices, nil
}

func (c *OutputRepository) CountAllVouchers(
	ctx context.Context,
) (uint64, error) {
	query := `SELECT COUNT(*) FROM vouchers`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("query error")
		return 0, err
	}
	defer stmt.Close()
	var countVoucher uint64
	err = stmt.GetContext(ctx, &countVoucher)
	if err != nil {
		return 0, err
	}
	return countVoucher, nil
}

func (c *OutputRepository) CountAllNotices(
	ctx context.Context,
) (uint64, error) {
	query := `SELECT COUNT(*) FROM notices`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("query error")
		return 0, err
	}
	defer stmt.Close()
	var countVoucher uint64
	err = stmt.GetContext(ctx, &countVoucher)
	if err != nil {
		return 0, err
	}
	return countVoucher, nil
}
