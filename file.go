package mageutil

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

func CopyInDir(destDir, srcDir string, files ...string) error {
	paths := make(map[string]string, len(files))
	for _, filename := range files {
		srcPath := filepath.Join(srcDir, filename)
		destPath := filepath.Join(destDir, filename)
		paths[srcPath] = destPath
	}
	return CopyFiles(paths)
}

func CopyFiles(paths map[string]string) error {
	for srcPath, destPath := range paths {
		newer, err := target.Path(destPath, srcPath)
		if err != nil {
			return fmt.Errorf("checking %s vs %s: %w", srcPath, destPath, err)
		}
		if newer {
			VerboseF("copying %s -> %s\n", srcPath, destPath)
			if err := sh.Copy(destPath, srcPath); err != nil {
				return fmt.Errorf("copying %s -> %s: %w", srcPath, destPath, err)
			}
		}

	}
	return nil
}

func CopyRecursively(destDir, srcDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("making source relative: %w", err)
		}
		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return Mkdir(destPath)
		}
		VerboseF("copying %s -> %s\n", path, destPath)
		return sh.Copy(destPath, path)
	})
}

func DirGlob(dir, glob string) ([]string, error) {
	matches := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		match, err := filepath.Match(glob, name)
		if err != nil {
			return nil, fmt.Errorf("bad glob: %w", err)
		}
		if !match {
			continue
		}
		matches = append(matches, name)
	}
	return matches, nil
}

func Mkdir(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("checking for %s: %w", path, err)
	}
	VerboseF("creating %s\n", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	return nil
}

func Newer(paths map[string]string) (bool, error) {
	for srcFile, destFile := range paths {
		newer, err := target.Path(destFile, srcFile)
		if err != nil {
			return false, fmt.Errorf("comparing %s -> %s: %w", srcFile, destFile, err)
		}
		if newer {
			return true, nil
		}
	}
	return false, nil
}

func ZipDir(inPath, outPath string) error {
	outfh, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating zip file: %w", err)
	}
	defer outfh.Close()

	writer := zip.NewWriter(outfh)

	inDir := os.DirFS(inPath)
	if err := writer.AddFS(inDir); err != nil {
		return fmt.Errorf("adding in path: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing zip: %w", err)
	}
	if err := outfh.Sync(); err != nil {
		return fmt.Errorf("syncing zip: %w", err)
	}

	return nil
}

func ZipFiles(out string, files map[string]string) error {
	outfh, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer outfh.Close()

	zw := zip.NewWriter(outfh)

	for src, dest := range files {
		if err := addToZip(zw, src, dest); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", out, err)
	}
	if err := outfh.Sync(); err != nil {
		return fmt.Errorf("syncing %s: %w", out, err)
	}
	return nil
}

func addToZip(zw *zip.Writer, src, dest string) error {
	w, err := zw.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s file: %w", dest, err)
	}
	infh, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s file: %w", src, err)
	}
	defer infh.Close()
	if _, err := io.Copy(w, infh); err != nil {
		return fmt.Errorf("writing %s file: %w", src, err)
	}
	return nil
}
