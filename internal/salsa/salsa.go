package salsa

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
)

const filePermission = 0755

type SalsaWorker struct {
	Address string
}

func (w SalsaWorker) String() string {
	return fmt.Sprintf("Salsa %s", w.Address)
}

func downloadSalsa(url string, destination string) (string, error) {
	// Creates temp file
	out, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Download files
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Escreve o conteúdo do download no arquivo
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return "", nil
}

func (w SalsaWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	url := "https://github.com/Calindra/salsa/releases/download/v1.0.10/salsa"
	tmpFile := "/tmp/salsa"

	// Verifica se o arquivo já existe em /tmp
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		// Arquivo não existe, faça o download
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

	// Dá permissão de execução ao arquivo temporário
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