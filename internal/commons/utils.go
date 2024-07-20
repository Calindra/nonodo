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
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
)

// Extract a tar.gz archive to a destination directory
func ExtractTarGz(archive []byte, destDir string) error {
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

// Extract a zip archive to a destination directory
func ExtractZip(archive []byte, destDir string) error {
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

// Get assets of latest release or prerelease from GitHub
func GetAssetsFromLastReleaseGitHub(ctx context.Context, client *github.Client, namespace, repository string) ([]ReleaseAsset, error) {
	// List the tags of the GitHub repository
	slog.Debug("Listing tags for", namespace, repository)

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
			slog.Debug("Asset", "tag", r.GetTagName(), "name", a.GetName(), "url", a.GetBrowserDownloadURL())
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