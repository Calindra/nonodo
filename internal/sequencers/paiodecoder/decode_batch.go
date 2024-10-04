package paiodecoder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/google/go-github/github"
)

const (
	DEFAULT_NAME_PROGRAM = "decode-batch"
)

func (d *DecodeBatchPaio) IsDecodeBatchInstalled() bool {
	location := path.Join(os.TempDir(), d.NameProgram)
	_, err := os.Stat(location)
	return err == nil
}

type DecodePaioConfig struct {
	AssetPaio   commons.ReleaseAsset `json:"asset_paio"`
	LatestCheck string               `json:"latest_check"`
}

func (a DecodeBatchPaio) TryLoadConfig() (*DecodePaioConfig, error) {
	root := filepath.Join(os.TempDir())
	file := filepath.Join(root, a.ConfigFilename)
	if _, err := os.Stat(file); err == nil {
		slog.Debug("paio: config already exists", "path", file)
		cfg, err := LoadPaioConfig(file)
		if err == nil {
			slog.Debug("paio: config is nightly, download new...", "tag", cfg.AssetPaio.Tag)
			return nil, nil
		}
		return cfg, err
	}
	slog.Debug("paio: config not found", "path", file)

	return nil, nil
}

func LoadPaioConfig(path string) (*DecodePaioConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("paio: failed to read config %s", err.Error())
	}

	var config DecodePaioConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("paio: failed to unmarshal config %s", err.Error())
	}

	return &config, nil
}

type DecodeBatchPaio struct {
	NameProgram    string
	Namespace      string
	Repository     string
	ConfigFilename string
	Client         *github.Client
}

func NewDecodeBatchPaio() commons.HandleRelease {
	return DecodeBatchPaio{
		NameProgram: DEFAULT_NAME_PROGRAM,
		// Change for Cartesi when available
		Namespace:      "Calindra",
		Repository:     "paio",
		ConfigFilename: "decode-batch.nonodo.json",
		Client:         github.NewClient(nil),
	}
}

// DownloadAsset implements commons.HandleRelease.
func (d DecodeBatchPaio) DownloadAsset(ctx context.Context, release *commons.ReleaseAsset) (string, error) {
	root := filepath.Join(os.TempDir(), release.Tag)
	var perm os.FileMode = 0755 | os.ModeDir
	err := os.MkdirAll(root, perm)
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir %s", err.Error())
	}

	filename := "decode-batch"
	if runtime.GOOS == commons.WINDOWS {
		filename = "decode-batch.exe"
	}

	decodeExec := filepath.Join(root, filename)
	slog.Debug("executable", "path", decodeExec)
	if _, err := os.Stat(decodeExec); err == nil {
		slog.Debug("executable already downloaded", "path", decodeExec)
		return decodeExec, nil
	}

	slog.Debug("downloading", "id", release.AssetId, "to", root)

	rc, redirect, err := d.Client.Repositories.DownloadReleaseAsset(ctx, d.Namespace, d.Repository, release.AssetId)
	if err != nil {
		return "", fmt.Errorf("failed to download asset %s", err.Error())
	}

	if redirect != "" {
		slog.Debug("redirect asset", "url", redirect)

		res, err := http.Get(redirect)
		if err != nil {
			return "", fmt.Errorf("failed to download asset %s", err.Error())
		}

		rc = res.Body
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read asset %s", err.Error())
	}

	slog.Debug("downloaded compacted file")

	err = d.ExtractAsset(data, release.Filename, root)
	if err != nil {
		return "", fmt.Errorf("failed to extract asset %s", err.Error())
	}

	release.Path = root

	// Save path on config
	// cfg := NewpaioConfig(*release)
	// err = d.SaveConfigOnDefaultLocation(cfg)
	// if err != nil {
	// 	return "", err
	// }

	return decodeExec, nil
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
	platform, err := d.PlatformCompatible()
	if err != nil {
		return nil, err
	}
	slog.Debug("paio:", "platform", platform)

	config, err := d.TryLoadConfig()
	if err != nil {
		return nil, err
	}

	paioTag, fromEnv := os.LookupEnv("PAIO_TAG")

	slog.Debug("using", "tag", paioTag, "fromEnv", fromEnv)

	var target *commons.ReleaseAsset = nil

	// Get release asset from config
	if config != nil {
		// Show config
		cfgStr, err := json.Marshal(config)
		if err != nil {
			slog.Debug("paio:", "config", config)
		} else {
			slog.Debug("paio:", "config", string(cfgStr))
		}

		if config.AssetPaio.Tag == paioTag {
			target = &config.AssetPaio
			return target, nil
		}
	}

	assets, err := commons.GetAssetsFromLastReleaseGitHub(ctx, d.Client, d.Namespace, d.Repository, paioTag)
	if err != nil {
		return nil, err
	}

	for _, paioAssets := range assets {
		if paioAssets.Filename == platform {
			target = &paioAssets
			break
		}
	}

	targetStr, err := json.Marshal(target)
	if err != nil {
		slog.Debug("paio:", "target", target)
	} else {
		slog.Debug("paio:", "target", string(targetStr))
	}

	if target != nil {
		// c := NewpaioConfig(*target)
		// err := d.SaveConfigOnDefaultLocation(c)
		// if err != nil {
		// return nil, err
		// }

		return target, nil
	}

	return nil, fmt.Errorf("paio: no compatible release found")
}

// ListRelease implements commons.HandleRelease.
func (d DecodeBatchPaio) ListRelease(ctx context.Context) ([]commons.ReleaseAsset, error) {
	return commons.GetAssetsFromLastReleaseGitHub(ctx, d.Client, d.Namespace, d.Repository, "")
}

// PlatformCompatible implements commons.HandleRelease.
func (d DecodeBatchPaio) PlatformCompatible() (string, error) {
	// Check if the platform is compatible
	slog.Debug("paio: System", "GOARCH:", runtime.GOARCH, "GOOS:", runtime.GOOS)
	goarch := runtime.GOARCH
	goos := runtime.GOOS

	if ((goarch == "amd64") && (goos == commons.WINDOWS || goos == "linux")) ||
		((goarch == "amd64" || goarch == "arm64") && goos == "darwin") {
		return d.FormatNameRelease("", goos, goarch, ""), nil
	}

	return "", fmt.Errorf("paio: platform not supported: os = %s; arch = %s", goarch, goos)
}

// Prerequisites implements commons.HandleRelease.
func (d DecodeBatchPaio) Prerequisites(ctx context.Context) error {
	return nil
}
