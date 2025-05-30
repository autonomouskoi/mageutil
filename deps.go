package mageutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// HasExec returns an error if any of cmds don't satisfy exec.LookPath
func HasExec(cmds ...string) error {
	missing := []string{}
	for _, cmd := range cmds {
		_, err := exec.LookPath(cmd)
		if err != nil {
			missing = append(missing, cmd)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return errors.New("missing tools: " + strings.Join(missing, ", "))
	}
	return nil
}

// HasFiles returns an error if any of files don't exist
func HasFiles(files ...string) error {
	missing := []string{}
	for _, testPath := range files {
		_, err := os.Stat(testPath)
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("checking %s: %w", testPath, err)
		}
		missing = append(missing, testPath)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return errors.New("missing files: " + strings.Join(missing, ", "))
	}
	return nil
}
