package topolib

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
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

var (
	errNoTargetDir = errors.New("cannot find a target dir")
)

type fsUpdater struct {
	ctx        context.Context
	cancel     context.CancelFunc
	logger     Logger
	provider   OfflineProvider
	usageStats *UsageStats
}

func (f *fsUpdater) Name() string {
	return f.provider.Name()
}

func (f *fsUpdater) Lookup(ctx context.Context, ip net.IP) (ProviderLookupResult, error) {
	return f.provider.Lookup(ctx, ip)
}

func (f *fsUpdater) Start() error {
	if err := f.doInitialCleaning(); err != nil {
		return fmt.Errorf("cannot do an initial cleaning: %w", err)
	}

	targetDir, err := f.getTargetDir()

	switch {
	case err == nil:
		if err := f.provider.Open(targetDir); err != nil {
			return fmt.Errorf("cannot open a directory %s: %w", targetDir, err)
		}
	case !errors.Is(err, errNoTargetDir):
		return fmt.Errorf("cannot detect target dir: %w", err)
	}

	go f.bgUpdate()

	return nil
}

func (f *fsUpdater) Shutdown() {
	f.cancel()
	f.provider.Shutdown()
}

func (f *fsUpdater) doInitialCleaning() error {
	rootDir := f.provider.BaseDirectory()

	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return fmt.Errorf("cannot read a base directory: %w", err)
	}

	targetDirs := []string{}
	toDelete := []string{}

	for _, v := range infos {
		fullPath := filepath.Join(rootDir, v.Name())

		if v.IsDir() && strings.HasPrefix(v.Name(), FsTargetDirPrefix) {
			targetDirs = append(targetDirs, fullPath)
		} else {
			toDelete = append(toDelete, fullPath)
		}
	}

	// if we have more than a single target directory, it is a time to
	// drop them all and start from scratch.
	if len(targetDirs) > 1 {
		toDelete = append(toDelete, targetDirs...)
	}

	for _, v := range toDelete {
		if err := os.RemoveAll(v); err != nil {
			return fmt.Errorf("cannot delete %s: %w", v, err)
		}
	}

	return nil
}

func (f *fsUpdater) bgUpdate() {
	timer := time.NewTicker(f.provider.UpdateEvery())
	defer timer.Stop()

	if err := f.doUpdate(); err != nil {
		f.logger.UpdateError(f.Name(), err)
	} else {
		f.usageStats.Updated()
		f.logger.UpdateInfo(f.Name(), "db has been updated")
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-timer.C:
			if err := f.doUpdate(); err != nil {
				f.logger.UpdateError(f.Name(), err)
			} else {
				f.logger.UpdateInfo(f.Name(), "db has been updated")
			}
		}
	}
}

func (f *fsUpdater) doUpdate() error {
	currentTargetDir, err := f.getTargetDir()
	if err != nil && !errors.Is(err, errNoTargetDir) {
		return fmt.Errorf("cannot detect current target dir: %w", err)
	}

	rootDir := f.provider.BaseDirectory()

	tmpDir, err := ioutil.TempDir(rootDir, "")
	if err != nil {
		return fmt.Errorf("cannot create a temporary directory: %w", err)
	}

	defer os.RemoveAll(tmpDir) // nolint: errcheck

	if err := f.provider.Download(f.ctx, tmpDir); err != nil {
		return fmt.Errorf("cannot download to tmp directory: %w", err)
	}

	targetDirName, err := f.getTargetDirName(tmpDir)
	if err != nil {
		return fmt.Errorf("cannot get a target dir name: %w", err)
	}

	if targetDirName == currentTargetDir {
		return nil
	}

	if currentTargetDir != "" {
		if err := os.RemoveAll(currentTargetDir); err != nil {
			return fmt.Errorf("cannot remove current target dir: %w", err)
		}
	}

	if err := os.Rename(tmpDir, targetDirName); err != nil {
		return fmt.Errorf("cannot rename tmp dir to target one: %w", err)
	}

	if err := f.provider.Open(targetDirName); err != nil {
		return fmt.Errorf("cannot open a target dir: %w", err)
	}

	return nil
}

func (f *fsUpdater) getTargetDir() (string, error) {
	rootDir := f.provider.BaseDirectory()

	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return "", fmt.Errorf("cannot read base directory: %w", err)
	}

	for _, v := range infos {
		if v.IsDir() && strings.HasPrefix(v.Name(), FsTargetDirPrefix) {
			return filepath.Join(rootDir, v.Name()), nil
		}
	}

	return "", errNoTargetDir
}

func (f *fsUpdater) getTargetDirName(rootDir string) (string, error) {
	hasher := sha256.New()
	startSign := []byte{0}
	fileSign := []byte{1}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case info.IsDir():
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("cannot build a relative path of %s to %s: %w",
				path,
				rootDir,
				err)
		}

		hasher.Write(startSign)         // nolint: errcheck
		io.WriteString(hasher, relPath) // nolint: errcheck

		fp, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open a file %s: %w", path, err)
		}

		defer fp.Close()

		hasher.Write(fileSign) // nolint: errcheck
		io.Copy(hasher, fp)    // nolint: errcheck

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("cannot calculate a checksum: %w", err)
	}

	baseName := FsTargetDirPrefix + hex.EncodeToString(hasher.Sum(nil))

	return filepath.Join(f.provider.BaseDirectory(), baseName), nil
}
