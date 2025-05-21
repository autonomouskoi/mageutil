package mageutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

// GoProto builds .pb.go code from a .proto
func GoProto(dest, src, out, include, opt string) error {
	newer, err := target.Path(dest, src)
	if err != nil {
		return fmt.Errorf("testing %s vs %s: %w", src, dest, err)
	}
	if !newer {
		return nil
	}
	VerboseF("protoc %s -> %s\n", src, dest)
	err = sh.Run("protoc",
		"-I", include,
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
func GoProtosInDir(dir, include, opt string) error {
	protos, err := DirGlob(dir, "*.proto")
	if err != nil {
		return fmt.Errorf("matchng files: %w", err)
	}
	for _, srcPath := range protos {
		destPath := strings.TrimSuffix(srcPath, ".proto") + ".pb.go"
		if err := GoProto(destPath, filepath.Join(dir, srcPath), dir, include, opt); err != nil {
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

var pcggl string
var pcgglOnce sync.Once

func TinyGoProto(
	dstPath string, // the final .pb.go file
	srcPath string, // the .proto file
	includeDir string, // for -I
) error {
	newer, err := target.Path(dstPath, srcPath)
	if err != nil {
		return fmt.Errorf("testing %s vs %s: %w", srcPath, dstPath, err)
	}
	if !newer {
		return nil
	}

	pcgglOnce.Do(func() {
		var err error
		pcggl, err = exec.LookPath("protoc-gen-go-lite")
		if err != nil {
			pcggl = ""
			return
		}
	})

	dstDir := filepath.Join(filepath.Dir(dstPath), "protoc-out")
	if err := Mkdir(dstDir); err != nil {
		return fmt.Errorf("creating %s: %w", dstDir, err)
	}
	defer sh.Rm(dstDir)

	err = sh.Run("protoc",
		"--plugin", pcggl,
		"--go-lite_opt", "features=marshal+unmarshal+size+equal+clone",
		"-I", includeDir,
		"--go-lite_out", dstDir,
		srcPath,
	)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(dstDir, func(path string, d fs.DirEntry, _ error) error {
		if !strings.HasSuffix(d.Name(), ".pb.go") {
			return nil
		}
		outfh, err := os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", dstPath, err)
		}
		defer outfh.Close()

		inFile := path
		infh, err := os.Open(inFile)
		if err != nil {
			return fmt.Errorf("opening %s: %w", inFile, err)
		}
		defer infh.Close()

		if _, err := io.Copy(outfh, infh); err != nil {
			return fmt.Errorf("copying %s -> %s: %w", inFile, dstPath, err)
		}
		if err := outfh.Sync(); err != nil {
			return fmt.Errorf("syncing %s: %w", dstPath, err)
		}
		return filepath.SkipAll
	})
	if err != nil {
		return fmt.Errorf("copying generated proto: %w", err)
	}

	return nil
}
