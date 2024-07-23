package commons

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
)

// ReleaseAsset represents a release asset from GitHub
type ReleaseAsset struct {
	Tag      string `json:"tag"`
	AssetId  int64  `json:"asset_id"`
	Filename string `json:"filename"`
	Url      string `json:"url"`
	Path     string `json:"path"`
}

// Interface for handle libraries on GitHub
type HandleRelease interface {
	FormatNameRelease(prefix, goos, goarch, version string) string
	PlatformCompatible() (string, error)
	ListRelease(ctx context.Context) ([]ReleaseAsset, error)
	GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error)
	DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error)
	ExtractAsset(archive []byte, filename string, destDir string) error
}

// Anvil implementation from HandleRelease
type AnvilRelease struct {
	Namespace  string
	Repository string
	Client     *github.Client
}

func NewAnvilRelease() HandleRelease {
	return &AnvilRelease{
		Namespace:  "foundry-rs",
		Repository: "foundry",
		Client:     github.NewClient(nil),
	}
}

// FormatNameRelease implements HandleRelease.
func (a AnvilRelease) FormatNameRelease(_, goos, goarch, _ string) string {
	ext := ".tar.gz"
	myos := goos

	if goos == "windows" {
		ext = ".zip"
		myos = "win32"
	}
	return "foundry_nightly_" + myos + "_" + goarch + ext
}

// PlatformCompatible implements HandleRelease.
func (a AnvilRelease) PlatformCompatible() (string, error) {
	// Check if the platform is compatible with Anvil
	slog.Debug("System", "GOARCH:", runtime.GOARCH, "GOOS:", runtime.GOOS)
	goarch := runtime.GOARCH
	goos := runtime.GOOS

	if (goarch == "amd64" && goos == "windows") ||
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
		return fmt.Errorf("format unsupported: %s", filename)
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

	anvilExec := filepath.Join(root, "anvil")
	slog.Debug("Anvil executable", "path", anvilExec)
	if _, err := os.Stat(anvilExec); err == nil {
		slog.Debug("Anvil already downloaded", "path", anvilExec)
		return anvilExec, nil
	}

	slog.Debug("Downloading anvil", "id", release.AssetId, "to", root)

	rc, redirect, err := a.Client.Repositories.DownloadReleaseAsset(ctx, a.Namespace, a.Repository, release.AssetId)

	if err != nil {
		return "", fmt.Errorf("anvil: failed to download asset %s", err.Error())
	}

	if redirect != "" {
		slog.Debug("Redirect", "url", redirect)

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

	slog.Debug("Downloaded compacted file anvil")

	err = a.ExtractAsset(data, release.Filename, root)

	if err != nil {
		return "", fmt.Errorf("anvil: failed to extract asset %s", err.Error())
	}

	release.Path = root

	return anvilExec, nil
}

// ListRelease implements HandleRelease.
func (a *AnvilRelease) ListRelease(ctx context.Context) ([]ReleaseAsset, error) {
	return GetAssetsFromLastReleaseGitHub(ctx, a.Client, a.Namespace, a.Repository)
}

// GetLatestReleaseCompatible implements HandleRelease.
func (a *AnvilRelease) GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error) {
	p, err := a.PlatformCompatible()
	if err != nil {
		return nil, err
	}

	assets, err := GetAssetsFromLastReleaseGitHub(ctx, a.Client, a.Namespace, a.Repository)
	if err != nil {
		return nil, err
	}

	for _, a := range assets {
		if a.Filename == p {
			return &a, nil
		}
	}

	return nil, fmt.Errorf("anvil: no compatible release found")
}
