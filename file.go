package mageutil

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

// CopyInDir copies files from srcDir to destDir
func CopyInDir(destDir, srcDir string, files ...string) error {
	paths := make(map[string]string, len(files))
	for _, filename := range files {
		srcPath := filepath.Join(srcDir, filename)
		destPath := filepath.Join(destDir, filename)
		paths[srcPath] = destPath
	}
	return CopyFiles(paths)
}

// CopyFiles paths with keys as source and values as destinations
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

// CopyRecursively recursively copies srcDir to destDir
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

// DirGlob performs a glob operation on dir. The returned matches do not have
// dir as a prefix.
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

// Mkdir calls os.MkdirAll as needed to ensure a directory exists
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

// Newer considers paths as a map of file path keys to file path values. It
// returns true if any key path is newer than its corresponding value path.
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

// ZipDir writes a zip file containing all the files in inPath
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

// ZipFiles creates a zip file including files where the keys are paths to
// files to include and values are how they should be named in the zip.
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

// SyncDirBasic sync the contents of srcDir to destDir, creating destDir as
// needed. When a file or directory exists in destDir but not srcDir it is
// deleted from srcDir. If the size of a file from source matches the size of
// the corresponding file in dest and the mod time of the dest file is the same
// or newer than the src file, the file is not copied. SyncDirBasic handles only
// files and directories, doesn't set timestamps or file permissions
func SyncDirBasic(srcDir, destDir string) error {
	srcEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcDir, err)
	}
	srcMap := make(map[string]os.DirEntry, len(srcEntries))
	for i, entry := range srcEntries {
		srcMap[entry.Name()] = srcEntries[i]
	}

	destEntries, err := os.ReadDir(destDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading %s: %w", destDir, err)
		}
		if err := Mkdir(destDir); err != nil {
			return fmt.Errorf("creating %s: %w", destDir, err)
		}
	}
	destMap := make(map[string]os.DirEntry, len(destEntries))
	for i, entry := range destEntries {
		destMap[entry.Name()] = destEntries[i]
	}

	// delete unwanted
	for name := range destMap {
		if _, ok := srcMap[name]; !ok {
			deletePath := filepath.Join(destDir, name)
			if err := sh.Rm(deletePath); err != nil {
				return fmt.Errorf("deleting %s: %w", deletePath, err)
			}
		}
	}

	// compare wanted
	for name, entry := range srcMap {
		srcPath := filepath.Join(srcDir, name)
		destPath := filepath.Join(destDir, name)
		if entry.IsDir() {
			if destEntry, present := destMap[name]; present && !destEntry.IsDir() {
				if err := sh.Rm(destPath); err != nil {
					return fmt.Errorf("deleting non-dir %s: %w", destPath, err)
				}
			}
			if err := SyncDirBasic(srcPath, destPath); err != nil {
				return fmt.Errorf("syncing %s -> %s: %w", srcPath, destPath, err)
			}
			continue
		}

		srcStat, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("statting %s: %w", srcPath, err)
		}
		doCopy := false
		if destStat, err := os.Stat(destPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("stating %s: %w", destPath, err)
			}
			doCopy = true // not existing is okay
		} else if destStat.IsDir() { // src is not a dir
			if err := sh.Rm(destPath); err != nil {
				return fmt.Errorf("deleting non-file %s: %w", destPath, err)
			}
			doCopy = true
		} else if srcStat.Size() != destStat.Size() {
			doCopy = true
		} else if srcStat.ModTime().After(destStat.ModTime()) {
			doCopy = true
		}
		if doCopy {
			if err := sh.Copy(destPath, srcPath); err != nil {
				return fmt.Errorf("copying %s -> %s: %w", srcPath, destPath, err)
			}
		}
	}

	return nil
}

func ReplaceInFile(filepath, from, to string) error {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	replaced := bytes.ReplaceAll(b, []byte(from), []byte(to))
	if err := os.WriteFile(filepath, replaced, 0); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}
