package commons

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/go-github/github"
)

type ReleaseAsset struct {
	Tag      string `json:"tag"`
	Id       int64  `json:"id"`
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

type HandleRelease interface {
	Format(prefix, goos, goarch, version string) string
	PlatformCompatible() (string, error)
	ListRelease(ctx context.Context) ([]ReleaseAsset, error)
	GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error)
	DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error)
}

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

// Format implements HandleRelease.
func (a AnvilRelease) Format(_, goos, goarch, _ string) string {
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

	if goarch == "amd64" && goos == "windows" {
		return a.Format("", goarch, goarch, ""), nil
	}

	if (goarch == "amd64" || goarch == "arm64") && (goos == "linux" || goos == "darwin") {
		return a.Format("", goos, goarch, ""), nil
	}

	return "", fmt.Errorf("anvil: platform not supported")
}

// DownloadAsset implements HandleRelease.
func (a *AnvilRelease) DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error) {
	tmp, err := os.MkdirTemp("", release.Tag)

	if err != nil {
		return "", fmt.Errorf("anvil: failed to create temp dir %s", err.Error())
	}

	location := filepath.Join(tmp, release.Filename)

	slog.Info("Downloading asset", "id", release.Id, "to", location)

	rc, redirect, err := a.Client.Repositories.DownloadReleaseAsset(ctx, a.Namespace, a.Repository, release.Id)

	if redirect != "" {
		slog.Info("Redirect", "url", redirect)
		defer rc.Close()
	}

	if err != nil {
		return "", fmt.Errorf("anvil: failed to download asset %s", err.Error())
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to read asset %s", err.Error())
	}

	slog.Info("Downloaded asset", "size", len(data))

	var permission fs.FileMode = 0644
	err = os.WriteFile(tmp, data, permission)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to write asset %s", err.Error())
	}

	return location, nil
}

// ListRelease implements HandleRelease.
func (a *AnvilRelease) ListRelease(ctx context.Context) ([]ReleaseAsset, error) {
	return getAssetsFromLastReleaseGitHub(ctx, a.Client, a.Namespace, a.Repository)
}

// GetLatestReleaseCompatible implements HandleRelease.
func (a *AnvilRelease) GetLatestReleaseCompatible(ctx context.Context) (*ReleaseAsset, error) {
	p, err := a.PlatformCompatible()
	if err != nil {
		return nil, err
	}

	assets, err := getAssetsFromLastReleaseGitHub(ctx, a.Client, a.Namespace, a.Repository)
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

func getAssetsFromLastReleaseGitHub(ctx context.Context, client *github.Client, namespace, repository string) ([]ReleaseAsset, error) {
	// List the tags of the GitHub repository
	slog.Info("Listing tags for", namespace, repository)

	releases, _, err := client.Repositories.ListReleases(ctx, namespace, repository, &github.ListOptions{
		PerPage: 1,
	})
	// release, _, err := client.Repositories.GetLatestRelease(ctx, namespace, repository)

	if err != nil {
		return nil, fmt.Errorf("%s(%s): failed to list releases %s", namespace, repository, err.Error())
	}

	fv := make([]ReleaseAsset, 0)

	for _, r := range releases {
		for _, a := range r.Assets {
			slog.Info("Asset", "name", a.GetName(), "url", a.GetBrowserDownloadURL())
			fv = append(fv, ReleaseAsset{
				Tag:      r.GetTagName(),
				Id:       a.GetID(),
				Filename: a.GetName(),
				Url:      a.GetBrowserDownloadURL(),
			})
		}
	}

	return fv, nil
}
