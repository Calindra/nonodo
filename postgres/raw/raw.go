package raw

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
)

func GetFilePath(name string) (string, error) {
	_, xdir, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current directory")
	}

	dir := filepath.Dir(xdir)

	slog.Debug("current directory", "dir", dir)

	filePath := path.Join(dir, name)

	slog.Debug("file path", "path", filePath)

	return filePath, nil
}

func RunDockerCompose(ctx context.Context) error {
	filePath, err := GetFilePath("compose.yml")
	if err != nil {
		return err
	}

	slog.Debug("docker compose file path", "path", filePath)

	// check if docker compose command is available
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose not found: %s", err)
	}
	slog.Debug("docker compose version", "output", string(output))

	cmd = exec.CommandContext(ctx, "docker", "compose", "-f", filePath, "up", "--wait")
	output, err = cmd.CombinedOutput()

	if err != nil {
		slog.Debug("docker compose up failed", "output", string(output))
		return fmt.Errorf("docker compose up failed: %s", err)
	}

	return nil
}

func StopDockerCompose(ctx context.Context) error {
	filePath, err := GetFilePath("compose.yml")
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", filePath, "down")

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Debug("docker compose down failed", "output", string(output))
		return fmt.Errorf("docker compose down failed: %s", err)
	}

	return nil
}
