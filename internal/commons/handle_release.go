package commons

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
	"strings"
	"time"

	"github.com/google/go-github/github"
)

// ReleaseAsset represents a release asset from GitHub
type ReleaseAsset struct {
	Tag      string `json:"tag"`
	AssetId  int64  `json:"asset_id"`
	Filename string `json:"filename"`
	Url      string `json:"url"`
	Path     string `json:"path"`
	NodeId   string `json:"node_id"`
}

// Interface for handle libraries on GitHub
type HandleRelease interface {
	// Name basead on version, arch, os with prefix
	FormatNameRelease(prefix, goos, goarch, version string) string
	// Check if the platform is compatible with the library and return the name of the release
	PlatformCompatible() (string, error)
	// List all releases from the repository
	ListRelease(ctx context.Context) ([]ReleaseAsset, error)
	// Get the latest release compatible with the platform
	GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error)
	// Check prerequisites for the library
	Prerequisites(ctx context.Context) error
	// Download the asset from the release
	DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error)
	// Extract the asset from the archive
	ExtractAsset(archive []byte, filename string, destDir string) error
}

// Anvil implementation from HandleRelease
type AnvilRelease struct {
	Namespace      string
	Repository     string
	ConfigFilename string
	Client         *github.Client
}

type AnvilConfig struct {
	AssetAnvil  ReleaseAsset `json:"asset_anvil"`
	LatestCheck string       `json:"latest_check"`
}

const WINDOWS = "windows"

func NewAnvilRelease() HandleRelease {
	return &AnvilRelease{
		Namespace:      "foundry-rs",
		Repository:     "foundry",
		ConfigFilename: "anvil.nonodo.json",
		Client:         github.NewClient(nil),
	}
}

func NewAnvilConfig(ra ReleaseAsset) *AnvilConfig {
	return &AnvilConfig{
		AssetAnvil:  ra,
		LatestCheck: time.Now().Format(time.RFC3339),
	}
}

func LoadAnvilConfig(path string) (*AnvilConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("anvil: failed to read config %s", err.Error())
	}

	var config AnvilConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("anvil: failed to unmarshal config %s", err.Error())
	}

	return &config, nil
}

func (a *AnvilConfig) SaveAnvilConfig(path string) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("anvil: failed to marshal config %s", err.Error())
	}

	var permission fs.FileMode = 0644
	err = os.WriteFile(path, data, permission)
	if err != nil {
		return fmt.Errorf("anvil: failed to write config %s", err.Error())
	}

	return nil
}

func (a AnvilRelease) SaveConfigOnDefaultLocation(config *AnvilConfig) error {
	root := filepath.Join(os.TempDir())
	file := filepath.Join(root, a.ConfigFilename)
	c := NewAnvilConfig(config.AssetAnvil)
	err := c.SaveAnvilConfig(file)
	return err
}

func (a AnvilRelease) TryLoadConfig() (*AnvilConfig, error) {
	root := filepath.Join(os.TempDir())
	file := filepath.Join(root, a.ConfigFilename)
	if _, err := os.Stat(file); err == nil {
		slog.Debug("anvil: config already exists", "path", file)
		cfg, err := LoadAnvilConfig(file)
		if err == nil && cfg.AssetAnvil.Tag == "nightly" {
			slog.Debug("anvil: config is nightly, download new...", "tag", cfg.AssetAnvil.Tag)
			return nil, nil
		}
		return cfg, err
	}
	slog.Debug("anvil: config not found", "path", file)

	return nil, nil
}

// FormatNameRelease implements HandleRelease.
func (a AnvilRelease) FormatNameRelease(_, goos, goarch, _ string) string {
	ext := ".tar.gz"
	myos := goos

	if goos == WINDOWS {
		ext = ".zip"
		myos = "win32"
	}
	return "foundry_nightly_" + myos + "_" + goarch + ext
}

// PlatformCompatible implements HandleRelease.
func (a AnvilRelease) PlatformCompatible() (string, error) {
	// Check if the platform is compatible with Anvil
	slog.Debug("anvil: System", "GOARCH:", runtime.GOARCH, "GOOS:", runtime.GOOS)
	goarch := runtime.GOARCH
	goos := runtime.GOOS

	if (goarch == "amd64" && goos == WINDOWS) ||
		((goarch == "amd64" || goarch == "arm64") && (goos == "linux" || goos == "darwin")) {
		return a.FormatNameRelease("", goos, goarch, ""), nil
	}

	return "", fmt.Errorf("anvil: platform not supported: os = %s; arch = %s", goarch, goos)
}

func (a *AnvilRelease) ExtractAsset(archive []byte, filename string, destDir string) error {
	if strings.HasSuffix(filename, ".zip") {
		return ExtractZip(archive, destDir)
	} else if strings.HasSuffix(filename, ".tar.gz") {
		return ExtractTarGz(archive, destDir)
	} else {
		return fmt.Errorf("anvil: format unsupported: %s", filename)
	}
}

// DownloadAsset implements HandleRelease.
func (a *AnvilRelease) DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error) {
	root := filepath.Join(os.TempDir(), release.Tag)
	var perm os.FileMode = 0755 | os.ModeDir
	err := os.MkdirAll(root, perm)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to create temp dir %s", err.Error())
	}

	filename := "anvil"
	if runtime.GOOS == WINDOWS {
		filename = "anvil.exe"
	}

	anvilExec := filepath.Join(root, filename)
	slog.Debug("anvil: executable", "path", anvilExec)
	if _, err := os.Stat(anvilExec); err == nil {
		slog.Debug("anvil: executable already downloaded", "path", anvilExec)
		return anvilExec, nil
	}

	slog.Debug("anvil: downloading", "id", release.AssetId, "to", root)

	rc, redirect, err := a.Client.Repositories.DownloadReleaseAsset(ctx, a.Namespace, a.Repository, release.AssetId)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to download asset %s", err.Error())
	}

	if redirect != "" {
		slog.Debug("anvil: redirect asset", "url", redirect)

		res, err := http.Get(redirect)
		if err != nil {
			return "", fmt.Errorf("anvil: failed to download asset %s", err.Error())
		}

		rc = res.Body
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to read asset %s", err.Error())
	}

	slog.Debug("anvil: downloaded compacted file anvil")

	err = a.ExtractAsset(data, release.Filename, root)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to extract asset %s", err.Error())
	}

	release.Path = root

	// Save path on config
	cfg := NewAnvilConfig(*release)
	err = a.SaveConfigOnDefaultLocation(cfg)
	if err != nil {
		return "", err
	}

	return anvilExec, nil
}

// ListRelease implements HandleRelease.
func (a *AnvilRelease) ListRelease(ctx context.Context) ([]ReleaseAsset, error) {
	return GetAssetsFromLastReleaseGitHub(ctx, a.Client, a.Namespace, a.Repository, "")
}

// GetLatestReleaseCompatible implements HandleRelease.
func (a *AnvilRelease) GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error) {
	platform, err := a.PlatformCompatible()
	if err != nil {
		return nil, err
	}
	slog.Debug("anvil:", "platform", platform)

	config, err := a.TryLoadConfig()
	if err != nil {
		return nil, err
	}

	anvilTag, fromEnv := os.LookupEnv("ANVIL_TAG")

	if !fromEnv {
		// Same behavior of Foundryup
		anvilTag = "nightly"
	}
	slog.Debug("anvil: using", "tag", anvilTag, "fromEnv", fromEnv)

	var target *ReleaseAsset = nil

	// Get release asset from config
	if config != nil {
		// Show config
		cfgStr, err := json.Marshal(config)
		if err != nil {
			slog.Debug("anvil:", "config", config)
		} else {
			slog.Debug("anvil:", "config", string(cfgStr))
		}

		if config.AssetAnvil.Tag == anvilTag {
			target = &config.AssetAnvil
			return target, nil
		}
	}

	assets, err := GetAssetsFromLastReleaseGitHub(ctx, a.Client, a.Namespace, a.Repository, anvilTag)
	if err != nil {
		return nil, err
	}

	for _, anvilAssets := range assets {
		if anvilAssets.Filename == platform {
			target = &anvilAssets
			break
		}
	}

	targetStr, err := json.Marshal(target)
	if err != nil {
		slog.Debug("anvil:", "target", target)
	} else {
		slog.Debug("anvil:", "target", string(targetStr))
	}

	if target != nil {
		c := NewAnvilConfig(*target)
		err := a.SaveConfigOnDefaultLocation(c)
		if err != nil {
			return nil, err
		}

		return target, nil
	}

	return nil, fmt.Errorf("anvil: no compatible release found")
}
