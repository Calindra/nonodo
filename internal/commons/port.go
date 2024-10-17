package commons

import (
	"fmt"
	"os/exec"
	"strings"
)

func IsPortInUse(port int) bool {
	// Construct the appropriate command based on the OS.
	cmd := exec.Command("bash", "-c", fmt.Sprintf("lsof -i :%d", port))

	// Execute the command and capture the output.
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "No such file") {
		return false
	}

	// If output is empty, the port is free.
	return len(output) > 0
}
