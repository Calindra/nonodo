package paiodecoder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/google/go-github/github"
)

const (
	DEFAULT_NAME_PROGRAM = "decode-batch"
)

func IsDecodeBatchInstalled() (string, bool) {
	filename := DEFAULT_NAME_PROGRAM
	if runtime.GOOS == commons.WINDOWS {
		filename = filename + ".exe"
	}
	location := filepath.Join(os.TempDir(), filename)
	_, err := os.Stat(location)
	return location, err == nil
}

type PaioConfig struct {
	AssetPaio   commons.ReleaseAsset `json:"asset_paio"`
	LatestCheck string               `json:"latest_check"`
}

func NewPaioConfig(ra commons.ReleaseAsset) *PaioConfig {
	return &PaioConfig{
		AssetPaio:   ra,
		LatestCheck: time.Now().Format(time.RFC3339),
	}
}

func (a *PaioConfig) SavePaioConfig(path string) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal config %s", err.Error())
	}

	var permission fs.FileMode = 0644
	err = os.WriteFile(path, data, permission)
	if err != nil {
		return fmt.Errorf("failed to write config %s", err.Error())
	}

	return nil
}

func (p PaioReleaseHandler) SaveConfigOnDefaultLocation(config *PaioConfig) error {
	root := filepath.Join(os.TempDir())
	file := filepath.Join(root, p.ConfigFilename)
	c := NewPaioConfig(config.AssetPaio)
	err := c.SavePaioConfig(file)
	return err
}

func (a PaioReleaseHandler) TryLoadConfig() (*PaioConfig, error) {
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

func LoadPaioConfig(path string) (*PaioConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("paio: failed to read config %s", err.Error())
	}

	var config PaioConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("paio: failed to unmarshal config %s", err.Error())
	}

	return &config, nil
}

type PaioReleaseHandler struct {
	Namespace      string
	Repository     string
	ConfigFilename string
	Client         *github.Client
}

func NewPaioReleaseHandler() commons.HandleRelease {
	return PaioReleaseHandler{
		// Change for Cartesi when available
		Namespace:      "cartesi",
		Repository:     "paio",
		ConfigFilename: "paio.nonodo.json",
		Client:         github.NewClient(nil),
	}
}

// DownloadAsset implements commons.HandleRelease.
func (d PaioReleaseHandler) DownloadAsset(ctx context.Context, release *commons.ReleaseAsset) (string, error) {
	folder := d.Repository + "-" + release.Tag
	root := filepath.Join(os.TempDir(), folder)
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

	err = d.ExtractAsset(data, filename, root)
	if err != nil {
		return "", fmt.Errorf("failed to extract asset %s", err.Error())
	}

	release.Path = root

	// Save path on config
	cfg := NewPaioConfig(*release)
	err = d.SaveConfigOnDefaultLocation(cfg)
	if err != nil {
		return "", err
	}
	slog.Debug("downloaded binary",
		"release.Filename", release.Filename,
		"decodeExec", decodeExec,
	)
	return decodeExec, nil
}

// ExtractAsset implements commons.HandleRelease.
func (d PaioReleaseHandler) ExtractAsset(archive []byte, filename string, destDir string) error {
	var permission fs.FileMode = 0755
	return os.WriteFile(filepath.Join(destDir, filename), archive, permission)
}

// FormatNameRelease implements commons.HandleRelease.
func (d PaioReleaseHandler) FormatNameRelease(prefix string, goos string, goarch string, version string) string {
	ext := ""
	myos := goos
	myarch := goarch

	if goos == commons.WINDOWS {
		ext = ".exe"
		myos = "windows-msvc"
	}

	if goos == "linux" {
		myos = "unknown-linux-gnu"
	}

	if goos == "darwin" {
		myos = "apple-darwin"
	}

	if goarch == "arm64" {
		myarch = "aarch64"
	}

	if goarch == commons.X86_64 {
		myarch = "x86_64"
	}

	return "decode-batch-" + myarch + "-" + myos + ext
}

// GetLatestReleaseCompatible implements commons.HandleRelease.
func (p PaioReleaseHandler) GetLatestReleaseCompatible(ctx context.Context) (*commons.ReleaseAsset, error) {
	platform, err := p.PlatformCompatible()
	if err != nil {
		return nil, err
	}
	slog.Debug("paio:", "platform", platform)

	config, err := p.TryLoadConfig()
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

	assets, err := commons.GetAssetsFromLastReleaseGitHub(ctx, p.Client, p.Namespace, p.Repository, paioTag)
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
		c := NewPaioConfig(*target)
		err := p.SaveConfigOnDefaultLocation(c)
		if err != nil {
			return nil, err
		}

		return target, nil
	}

	return nil, fmt.Errorf("paio: no compatible release found")
}

// ListRelease implements commons.HandleRelease.
func (d PaioReleaseHandler) ListRelease(ctx context.Context) ([]commons.ReleaseAsset, error) {
	return commons.GetAssetsFromLastReleaseGitHub(ctx, d.Client, d.Namespace, d.Repository, "")
}

// PlatformCompatible implements commons.HandleRelease.
func (d PaioReleaseHandler) PlatformCompatible() (string, error) {
	// Check if the platform is compatible
	slog.Debug("paio: System", "GOARCH:", runtime.GOARCH, "GOOS:", runtime.GOOS)
	goarch := runtime.GOARCH
	goos := runtime.GOOS

	if ((goarch == commons.X86_64) && (goos == commons.WINDOWS || goos == "linux")) ||
		((goarch == commons.X86_64 || goarch == "arm64") && goos == "darwin") {
		return d.FormatNameRelease("", goos, goarch, ""), nil
	}

	return "", fmt.Errorf("paio: platform not supported: os = %s; arch = %s", goarch, goos)
}

// Prerequisites implements commons.HandleRelease.
func (d PaioReleaseHandler) Prerequisites(ctx context.Context) error {
	return nil
}

func DownloadPaioDecoderExecutableAsNeeded() (string, error) {
	paioLocation, installed := IsDecodeBatchInstalled()
	if !installed {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		paioReleaseHandler := NewPaioReleaseHandler()
		location, err := commons.HandleReleaseExecution(ctx, paioReleaseHandler)
		if err != nil {
			slog.Error("error downloading paio batch decoder executable")
			return "", err
		}
		paioLocation = location
	}
	return paioLocation, nil
}
