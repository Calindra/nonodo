package commons

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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

type ReleaseAsset struct {
	Tag      string `json:"tag"`
	AssetId  int64  `json:"id"`
	Filename string `json:"filename"`
	Url      string `json:"url"`
	Path     string `json:"path"`
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

func (a *AnvilRelease) extractArchive(archive []byte, filename string, destDir string) error {
	if strings.HasSuffix(filename, ".zip") {
		return a.extractZip(archive, destDir)
	} else if strings.HasSuffix(filename, ".tar.gz") {
		return a.extractTarGz(archive, destDir)
	} else {
		return fmt.Errorf("formato de arquivo não suportado: %s", filename)
	}
}

func (a *AnvilRelease) extractTarGz(archive []byte, destDir string) error {
	gzipStream, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gzipStream.Close()

	tarReader := tar.NewReader(gzipStream)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		filePath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return err
			}
			destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer destFile.Close()

			_, err = io.Copy(destFile, tarReader)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unsupported type: %v", header.Typeflag)
		}
	}
	return nil
}

func (a *AnvilRelease) extractZip(archive []byte, destDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		filePath := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer destFile.Close()

		fileInArchive, err := file.Open()
		if err != nil {
			return err
		}
		defer fileInArchive.Close()

		_, err = io.Copy(destFile, fileInArchive)
		if err != nil {
			return err
		}
	}
	return nil
}

// DownloadAsset implements HandleRelease.
func (a *AnvilRelease) DownloadAsset(ctx context.Context, release *ReleaseAsset) (string, error) {
	tmp, err := os.MkdirTemp("", release.Tag)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to create temp dir %s", err.Error())
	}

	anvilExec := filepath.Join(tmp, "anvil")
	if _, err := os.Stat(anvilExec); err == nil {
		return tmp, nil
	}

	slog.Info("Downloading asset", "id", release.AssetId, "to", tmp)

	rc, redirect, err := a.Client.Repositories.DownloadReleaseAsset(ctx, a.Namespace, a.Repository, release.AssetId)

	if err != nil {
		return "", fmt.Errorf("anvil: failed to download asset %s", err.Error())
	}

	if redirect != "" {
		slog.Info("Redirect", "url", redirect)

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

	slog.Info("Downloaded asset")

	err = a.extractArchive(data, release.Filename, tmp)

	if err != nil {
		return "", fmt.Errorf("anvil: failed to extract asset %s", err.Error())
	}

	release.Path = tmp

	return anvilExec, nil
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

// Get assets of latest release or prerelease from GitHub
func getAssetsFromLastReleaseGitHub(ctx context.Context, client *github.Client, namespace, repository string) ([]ReleaseAsset, error) {
	// List the tags of the GitHub repository
	slog.Info("Listing tags for", namespace, repository)

	releases, _, err := client.Repositories.ListReleases(ctx, namespace, repository, &github.ListOptions{
		PerPage: 1,
	})

	// For stable releases
	// release, _, err := client.Repositories.GetLatestRelease(ctx, namespace, repository)

	if err != nil {
		return nil, fmt.Errorf("%s(%s): failed to list releases %s", namespace, repository, err.Error())
	}

	ra := make([]ReleaseAsset, 0)

	for _, r := range releases {
		for _, a := range r.Assets {
			slog.Info("Asset", "tag", r.GetTagName(), "name", a.GetName(), "url", a.GetBrowserDownloadURL())
			ra = append(ra, ReleaseAsset{
				Tag:      r.GetTagName(),
				AssetId:  a.GetID(),
				Filename: a.GetName(),
				Url:      a.GetBrowserDownloadURL(),
			})
		}
	}

	return ra, nil
}
