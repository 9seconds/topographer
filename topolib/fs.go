package topolib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// FsTargetDirPrefix is a prefix which is used to mark 'active'
	// directory with databases for the provider. All other
	// directories/files are ok to be removed at any given moment in
	// time.
	//
	// If there are many target directories, topographer uses a random
	// one.
	//
	// Suffix is generated based on a contents of the directory.
	// You can think about simplified merkle tree hash here.
	FsTargetDirPrefix = "target_"

	// FsTempDirPrefix defines a prefix for temporary directories populated
	// during update of the offline databases.
	//
	// It works in a following way:
	//    1. Each provider has its own base directory
	//    2. When time comes, topographer creates a new temporary
	//       and passes it to provider.
	//    3. Provider does some nasty things there: downloads files
	//       creates something and prepares a directory structure
	//       applicable for Open method
	//    4. Old target directory is removed and temporary one
	//       is renamed into a new target one.
	FsTempDirPrefix = "tmp_"
)

type fsDir struct {
	Dir string
}

func (f fsDir) TempDir() (string, error) {
	return ioutil.TempDir(f.Dir, FsTempDirPrefix)
}

func (f fsDir) GetTargetDir() (string, time.Time, error) {
	data, err := f.readDir()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("cannot find target dir: %w", err)
	}

	target := ""
	targetTime := time.Time{}

	for fullpath, info := range data {
		found := (info.IsDir() &&
			strings.HasPrefix(info.Name(), FsTargetDirPrefix) &&
			(targetTime.IsZero() || info.ModTime().Sub(targetTime) < 0))
		if found {
			target = fullpath
			targetTime = info.ModTime()
		}
	}

	return target, targetTime, nil
}

func (f fsDir) Cleanup(filesToSave ...string) error {
	data, err := f.readDir()
	if err != nil {
		return fmt.Errorf("cannot read current directory: %w", err)
	}

	for _, v := range filesToSave {
		delete(data, v)
	}

	for fullpath := range data {
		if err := os.RemoveAll(fullpath); err != nil {
			return fmt.Errorf("cannot remove %s: %w", fullpath, err)
		}
	}

	return nil
}

func (f fsDir) Promote(dir string) (string, bool, error) {
	checksum, err := f.makeChecksum(dir)
	if err != nil {
		return "", false, fmt.Errorf("cannot make a checksum: %w", err)
	}

	targetName := filepath.Join(f.Dir, FsTargetDirPrefix+checksum)

	if _, err := os.Stat(targetName); err == nil {
		return targetName, false, nil
	}

	if err := os.Rename(dir, targetName); err != nil {
		return "", false, fmt.Errorf("cannot rename %s to %s: %w", dir, targetName, err)
	}

	return targetName, true, nil
}

func (f fsDir) makeChecksum(dir string) (string, error) {
	hasher := sha256.New()
	newFileSign := []byte{0}
	fileContentsSign := []byte{1}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case info.IsDir():
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("cannot build a relative path of %s to %s: %w", path, dir, err)
		}

		hasher.Write(newFileSign)      // nolint: errcheck
		hasher.Write([]byte(relPath))  // nolint: errcheck
		hasher.Write(fileContentsSign) // nolint: errcheck

		fp, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open a file %s: %w", path, err)
		}

		defer fp.Close()

		if _, err := io.Copy(hasher, fp); err != nil {
			return fmt.Errorf("cannot copy a file contents of %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("cannot traverse directory %s: %w", dir, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (f fsDir) readDir() (map[string]os.FileInfo, error) {
	infos, err := ioutil.ReadDir(f.Dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read dir %s: %w", f.Dir, err)
	}

	rv := map[string]os.FileInfo{}

	for _, v := range infos {
		rv[filepath.Join(f.Dir, v.Name())] = v
	}

	return rv, nil
}
