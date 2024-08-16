package commons

import (
	"log/slog"
	"os"
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

func (d *DbFactory) CreateDb(pattern string) *sqlx.DB {
	file, err := os.CreateTemp(d.TempDir, pattern)
	if err != nil {
		slog.Error("Error creating temp file", "err", err)
		panic(err)
	}
	_, err = file.Write([]byte{})
	if err != nil {
		slog.Error("Error writing to temp file", "err", err)
		panic(err)
	}
	sqliteFileName := file.Name()
	file.Close()
	// db := sqlx.MustConnect("sqlite3", ":memory:")
	slog.Info("Creating db attempting", "sqliteFileName", sqliteFileName)
	return sqlx.MustConnect("sqlite3", sqliteFileName)
}

func (d *DbFactory) Cleanup() {
	if d.TempDir != "" {
		slog.Info("Cleaning up temp dir", "tempDir", d.TempDir)
		os.RemoveAll(d.TempDir)
	}
}
