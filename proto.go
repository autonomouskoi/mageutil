package mageutil

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

// GoProto builds .pb.go code from a .proto
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

// GoProtosInDir calls GoProto on all .proto files in dir
func GoProtosInDir(dir, opt string) error {
	protos, err := DirGlob(dir, "*.proto")
	if err != nil {
		return fmt.Errorf("matchng files: %w", err)
	}
	for _, srcPath := range protos {
		destPath := strings.TrimSuffix(srcPath, ".proto") + ".pb.go"
		if err := GoProto(destPath, filepath.Join(dir, srcPath), dir, opt); err != nil {
			return fmt.Errorf("running protoc on %s: %w", srcPath, err)
		}
	}
	return nil
}

// TSProtosInDir creates _pb.js files in destDir for all .proto files in srcDir
func TSProtosInDir(destDir, srcDir string) error {
	if err := HasExec("protoc"); err != nil {
		return err
	}
	plugin := filepath.Join(srcDir, "node_modules/.bin/protoc-gen-es")
	if runtime.GOOS == "windows" {
		plugin += ".cmd"
	}
	if err := HasFiles(plugin); err != nil {
		return err
	}
	protoDestDir := filepath.Join(destDir, "pb")
	if err := Mkdir(protoDestDir); err != nil {
		return fmt.Errorf("creating %s: %w", protoDestDir, err)
	}
	protos, err := DirGlob(srcDir, "*.proto")
	if err != nil {
		return fmt.Errorf("matching files: %w", err)
	}
	for _, srcFile := range protos {
		baseName := strings.TrimSuffix(filepath.Base(srcFile), ".proto")
		destFile := filepath.Join(protoDestDir, baseName+"_pb.js")
		srcFile = filepath.Join(srcDir, srcFile)
		newer, err := target.Path(destFile, srcFile)
		if err != nil {
			return fmt.Errorf("testing %s vs %s: %w", srcFile, destFile, err)
		}
		if !newer {
			continue
		}
		VerboseF("generating proto %s -> %s\n", srcFile, destFile)
		err = sh.Run("protoc",
			"--plugin", "protoc-gen-es="+plugin,
			"-I", srcDir,
			"--es_out", protoDestDir,
			srcFile,
		)
		if err != nil {
			return fmt.Errorf("generating proto %s -> %s\n: %w", srcFile, destFile, err)
		}
	}
	return nil
}
