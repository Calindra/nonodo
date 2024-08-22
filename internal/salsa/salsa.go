package salsa

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const filePermission = 0755
const amd64Archicteture = "amd64"

type SalsaWorker struct {
	Address string
}

func (w SalsaWorker) String() string {
	return fmt.Sprintf("Salsa %s", w.Address)
}

func downloadSalsa(url string, destination string) (string, error) {
	out, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return "", nil
}

func getBinary() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	var binary string

	switch os {
	case "linux":
		if arch == amd64Archicteture {
			binary = "salsa-linux-amd64"
		} else if arch == "arm64" {
			binary = "salsa-linux-arm64"
		}
	case "darwin": // macOS
		if arch == amd64Archicteture {
			binary = "salsa-macos-amd64"
		} else if arch == "arm64" {
			binary = "salsa-macos-arm64"
		}
	case "windows":
		if arch == amd64Archicteture {
			binary = "salsa-win32-amd64.exe"
		}
	default:
		binary = "unsupported"
	}

	return binary
}

func (w SalsaWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	binary := getBinary()

	if binary == "unsupported" {
		return fmt.Errorf("unsupported OS")
	}

	url := "https://github.com/Calindra/salsa/releases/download/v1.1.2/" + binary

	tempDir := os.TempDir()
	tmpFile := filepath.Join(tempDir, binary)

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		slog.Info("Downloading Salsa...")

		_, err := downloadSalsa(url, tmpFile)
		if err != nil {
			slog.Error("Error downloading Salsa: " + err.Error())
			return err
		}
		slog.Info("Salsa downloaded.")
	} else {
		slog.Warn("Salsa found. Skipping download.")
	}

	err := os.Chmod(tmpFile, filePermission)
	if err != nil {
		slog.Error("Error changing Salsa permissions: " + err.Error())
		return err
	}

	ready <- struct{}{}

	var cmd *exec.Cmd
	if w.Address != "" {
		cmd = exec.Command(tmpFile, "--address", w.Address)
	} else {
		cmd = exec.Command(tmpFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		slog.Error("Error executing Salsa: " + err.Error())
		return err
	}

	return nil
}
