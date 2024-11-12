// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package supervisor

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"regexp"
)

// Log the command output with the name as a prefix.
type commandLogger struct {
	name     string
	buffName string
	buffer   bytes.Buffer
}

func (w *commandLogger) Write(data []byte) (int, error) {
	_, err := w.buffer.Write(data)
	if err != nil {
		return 0, err
	}
	for {
		bytes, err := w.buffer.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return 0, err
		}
		line := string(bytes)
		line = line[:len(line)-1] // remove \n
		re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		line = re.ReplaceAllString(line, "") // remove color
		if len(line) > 0 {
			slog.Info("command: log", "command", w.name, "buffer", w.buffName,
				"line", line)
		}
	}
	return len(data), nil
}
