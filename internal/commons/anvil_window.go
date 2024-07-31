//go:build windows

package commons

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// Prerequisites implements HandleRelease.
func (a *AnvilRelease) Prerequisites(ctx context.Context) error {
	isInstalled, err := isVCRedistInstalled()
	if err != nil {
		return err
	}

	if isInstalled {
		return nil
	}

	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	fp, err := DownloadRuntime(runtimeCtx)
	if err != nil {
		return err
	}
	err = InstallRuntime(runtimeCtx, fp)
	if err != nil {
		return err
	}

	return nil
}

func InstallRuntime(ctx context.Context, path string) error {
	slog.Debug("Installing runtime", "path", path)

	cmd := exec.CommandContext(ctx, path, "/install", "/passive", "/norestart")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("anvil: failed to install runtime %s", err.Error())
	}

	return nil
}

// Download the runtime and install it.
// Get it from https://learn.microsoft.com/en-us/cpp/windows/latest-supported-vc-redist?view=msvc-170#latest-microsoft-visual-c-redistributable-version
func DownloadRuntime(ctx context.Context) (string, error) {
	url := "https://aka.ms/vs/17/release/vc_redist.x64.exe"
	slog.Debug("Downloading runtime", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to download runtime %s", err.Error())
	}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to download runtime %s", err.Error())
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to read runtime %s", err.Error())
	}

	root := filepath.Join(os.TempDir(), "foundry-runtime")
	var perm os.FileMode = 0755
	err = os.MkdirAll(root, perm|os.ModeDir)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to create temp dir %s", err.Error())
	}

	fp := filepath.Join(root, "vc_redist.x64.exe")
	err = os.WriteFile(fp, data, perm)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to write runtime %s", err.Error())
	}

	return fp, nil
}

func isVCRedistInstalled() (bool, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64`, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer key.Close()

	installed, _, err := key.GetIntegerValue("Installed")
	if err != nil {
		return false, err
	}

	return installed == 1, nil
}
