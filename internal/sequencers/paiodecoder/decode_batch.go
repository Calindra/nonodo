package paiodecoder

import (
	"context"
	"os"
	"path"

	"github.com/calindra/nonodo/internal/commons"
)

const (
	DEFAULT_NAME_PROGRAM = "decode-batch"
)

func (d *DecodeBatchPaio) IsDecodeBatchInstalled() bool {
	location := path.Join(os.TempDir(), d.NameProgram)
	_, err := os.Stat(location)
	return err == nil
}

type DecodeBatchPaio struct {
	NameProgram string
}

func NewDecodeBatchPaio(name string) commons.HandleRelease {
	return DecodeBatchPaio{
		NameProgram: DEFAULT_NAME_PROGRAM,
	}
}

// DownloadAsset implements commons.HandleRelease.
func (d DecodeBatchPaio) DownloadAsset(ctx context.Context, release *commons.ReleaseAsset) (string, error) {
	panic("unimplemented")
}

// ExtractAsset implements commons.HandleRelease.
func (d DecodeBatchPaio) ExtractAsset(archive []byte, filename string, destDir string) error {
	panic("unimplemented")
}

// FormatNameRelease implements commons.HandleRelease.
func (d DecodeBatchPaio) FormatNameRelease(prefix string, goos string, goarch string, version string) string {
	panic("unimplemented")
}

// GetLatestReleaseCompatible implements commons.HandleRelease.
func (d DecodeBatchPaio) GetLatestReleaseCompatible(ctx context.Context) (*commons.ReleaseAsset, error) {
	panic("unimplemented")
}

// ListRelease implements commons.HandleRelease.
func (d DecodeBatchPaio) ListRelease(ctx context.Context) ([]commons.ReleaseAsset, error) {
	panic("unimplemented")
}

// PlatformCompatible implements commons.HandleRelease.
func (d DecodeBatchPaio) PlatformCompatible() (string, error) {
	panic("unimplemented")
}

// Prerequisites implements commons.HandleRelease.
func (d DecodeBatchPaio) Prerequisites(ctx context.Context) error {
	panic("unimplemented")
}
