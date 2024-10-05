package commons

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/jmoiron/sqlx"
)

type DbFactory struct {
	TempDir string
	Timeout time.Duration
}

const TimeoutInSeconds = 10

func NewDbFactory() *DbFactory {
	tempDir, err := os.MkdirTemp("", "nonodo-test-*")
	if err != nil {
		slog.Error("Error creating temp dir", "err", err)
		panic(err)
	}

	return &DbFactory{
		TempDir: tempDir,
		Timeout: TimeoutInSeconds * time.Second,
	}
}

func (d *DbFactory) CreateTempDb() *sqlx.DB {
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	sqlitePath := path.Join(d.TempDir, sqliteFileName)
	slog.Info("Creating db attempting", "sqlitePath", sqlitePath)
	return sqlx.MustConnect("sqlite3", sqlitePath)
}

func (d *DbFactory) CreateDb(sqliteFileName string) *sqlx.DB {
	// db := sqlx.MustConnect("sqlite3", ":memory:")
	sqlitePath := path.Join(d.TempDir, sqliteFileName)
	slog.Info("Creating db attempting", "sqlitePath", sqlitePath)
	return sqlx.MustConnect("sqlite3", sqlitePath)
}

func (d *DbFactory) Cleanup() {
	if d.TempDir != "" {
		slog.Info("Cleaning up temp dir", "tempDir", d.TempDir)
		err := os.RemoveAll(d.TempDir)
		if err != nil {
			slog.Error("Error removing temp dir", "err", err)
		}
	}
}
