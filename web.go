package mageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

// BuildTypeScript builds all .ts files in srcDir to .js files in destDir, as
// needed
func BuildTypeScript(baseDir, srcDir, destDir string) error {
	dirEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("listing %s: %w", srcDir, err)
	}
	newer := false
	for _, entry := range dirEntries {
		if entry.Type() == os.ModeDir {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") {
			continue
		}
		baseName := strings.TrimSuffix(name, ".ts")
		destFile := filepath.Join(destDir, baseName+".js")
		srcFile := filepath.Join(srcDir, name)
		newer, err = target.Path(destFile, srcFile)
		if err != nil {
			return fmt.Errorf("testing %s vs %s: %w", srcFile, destFile, err)
		}
		if newer {
			break
		}
	}
	if newer {
		tsc := filepath.Join(baseDir, "node_modules", ".bin", "tsc")
		if runtime.GOOS == "windows" {
			tsc += ".cmd"
		}
		return sh.Run(tsc, "-p", filepath.Join(baseDir, "tsconfig.json"))
	}
	return nil
}
