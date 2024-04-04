package convenience

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type ConvenienceRepositoryImpl struct {
	db sqlx.DB
}

func (c *ConvenienceRepositoryImpl) CreateTables() error {
	schema := `CREATE TABLE vouchers (
		Destination text,
		Payload 	text,
		Executed	BOOLEAN,
		InputIndex 	integer,
		OutputIndex integer);`

	// execute a query on the server
	_, err := c.db.Exec(schema)
	return err
}

func (c *ConvenienceRepositoryImpl) CreateVoucher(
	ctx context.Context, voucher *ConvenienceVoucher,
) (*ConvenienceVoucher, error) {
	insertVoucher := `INSERT INTO vouchers (
		Destination,
		Payload,
		Executed,
		InputIndex,
		OutputIndex) VALUES (?, ?, ?, ?, ?)`
	c.db.MustExec(insertVoucher, voucher.Destination, voucher.Payload, voucher.Executed, voucher.InputIndex, voucher.OutputIndex)
	return nil, nil
}

func (c *ConvenienceRepositoryImpl) VoucherCount(
	ctx context.Context,
) (uint64, error) {
	var id int
	err := c.db.Get(&id, "SELECT count(*) FROM vouchers")
	if err != nil {
		return 0, nil
	}
	return uint64(id), nil
}
func (c *ConvenienceRepositoryImpl) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*ConvenienceVoucher, error) {
	query := `SELECT * FROM vouchers WHERE inputIndex = ? and outputIndex = ?`
	stmt, err := c.db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var p ConvenienceVoucher
	err = stmt.Get(&p, inputIndex, outputIndex)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
