package mageutil

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

func GoProto(dest, src, out, opt string) error {
	newer, err := target.Path(dest, src)
	if err != nil {
		return fmt.Errorf("testing %s vs %s: %w", src, dest, err)
	}
	if !newer {
		return nil
	}
	VerboseF("protoc %s -> %s\n", src, dest)
	err = sh.Run("protoc",
		"-I", out,
		"--go_out", out,
		"--go_opt", opt,
		src,
	)
	if err != nil {
		return fmt.Errorf("building %s -> %s: %w", src, dest, err)
	}
	return nil
}

func GoProtosInDir(dir, opt string) error {
	protos, err := DirGlob(dir, "*.proto")
	if err != nil {
		return fmt.Errorf("matchng files: %w", err)
	}
	for _, srcPath := range protos {
		destPath := filepath.Join(dir, strings.TrimSuffix(srcPath, ".proto")+".pb.go")
		if err := GoProto(destPath, srcPath, dir, opt); err != nil {
			return fmt.Errorf("running protoc on %s: %w", srcPath, err)
		}
	}
	return nil
}
