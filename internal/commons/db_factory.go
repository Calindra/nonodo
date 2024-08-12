package commons

import (
	"context"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
)

type DbFactory struct {
	TempDir string
	Timeout time.Duration
}

func NewDbFactory() *DbFactory {
	return &DbFactory{
		TempDir: "",
		Timeout: 5 * time.Second,
	}
}

func (d *DbFactory) CreateRootTmpDir() error {
	tempDir, err := os.MkdirTemp("", "nonodo-test-*")
	if err != nil {
		return err
	}
	d.TempDir = tempDir
	return nil
}

func (d *DbFactory) CreateDb(pattern string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()
	tempDir := d.TempDir
	sqlite, err := os.CreateTemp(tempDir, pattern)
	if err != nil {
		return nil, err
	}
	err = sqlite.Close()
	if err != nil {
		return nil, err
	}
	sqliteFileName := sqlite.Name()
	// db := sqlx.MustConnect("sqlite3", ":memory:")
	return sqlx.ConnectContext(ctx, "sqlite3", sqliteFileName)
}

func (d *DbFactory) Cleanup() error {
	return os.RemoveAll(d.TempDir)
}
