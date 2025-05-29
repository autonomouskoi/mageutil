package mageutil

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

var tinygoPath string
var tinygoOnce sync.Once

func TinyGoWASM(srcDir, outfile string) error {
	newer, err := target.Dir(outfile, srcDir)
	if err != nil {
		return fmt.Errorf("testing for newer files: %w", err)
	}
	if !newer {
		return nil
	}

	tinygoOnce.Do(func() {
		var err error
		tinygoPath, err = exec.LookPath("tinygo")
		if err != nil {
			tinygoPath = ""
			return
		}
	})
	if tinygoPath == "" {
		return errors.New("missing: tinygo")
	}

	return sh.Run(tinygoPath, "build",
		"-target", "wasi",
		"-buildmode", "c-shared",
		"-no-debug", "-scheduler=none", "-panic=trap", // reduce size
		"-o", outfile,
		srcDir,
	)
}
