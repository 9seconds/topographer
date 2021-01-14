package topolib

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/afero"
)

const (
	FsTargetDirPrefix = "target_"
	FsTempDirPrefix   = "tmp_"
)

var (
	errNoTargetDir = errors.New("cannot find a target dir")
)

type fsUpdater struct {
	ctx      context.Context
	cancel   context.CancelFunc
	logger   Logger
	provider OfflineProvider
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
		if err := f.provider.Open(f.getTargetFs(targetDir)); err != nil {
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
	baseFs := f.getBaseFs()

	infos, err := baseFs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("cannot read a base directory: %w", err)
	}

	targetDirs := []string{}
	toDelete := []string{}

	for _, v := range infos {
		if v.IsDir() && strings.HasPrefix(v.Name(), FsTargetDirPrefix) {
			targetDirs = append(targetDirs, v.Name())
		} else {
			toDelete = append(toDelete, v.Name())
		}
	}

	// if we have more than a single target directory, it is a time to
	// drop them all and start from scratch.
	if len(targetDirs) > 1 {
		toDelete = append(toDelete, targetDirs...)
	}

	for _, v := range toDelete {
		if err := baseFs.RemoveAll(v); err != nil {
			return fmt.Errorf("cannot delete %s: %w", v, err)
		}
	}

	return nil
}

func (f *fsUpdater) bgUpdate() {
	timer := time.NewTicker(f.provider.UpdateEvery())
	defer timer.Stop()

	baseFs := f.getBaseFs()

	if err := f.doUpdate(baseFs); err != nil {
		f.logger.UpdateError(f.Name(), err)
	} else {
		f.logger.UpdateInfo(f.Name(), "db has been updated")
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-timer.C:
			if err := f.doUpdate(baseFs); err != nil {
				f.logger.UpdateError(f.Name(), err)
			} else {
				f.logger.UpdateInfo(f.Name(), "db has been updated")
			}
		}
	}
}

func (f *fsUpdater) doUpdate(fs afero.Afero) error {
	currentTargetDir, err := f.getTargetDir()
	if err != nil && !errors.Is(err, errNoTargetDir) {
		return fmt.Errorf("cannot detect current target dir: %w", err)
	}

	tmpDir, err := fs.TempDir(".", FsTempDirPrefix)
	if err != nil {
		return fmt.Errorf("cannot create a temporary directory: %w", err)
	}

	defer fs.RemoveAll(tmpDir) // nolint: errcheck

	tmpFs := afero.Afero{
		Fs: afero.NewBasePathFs(fs, tmpDir),
	}

	if err := f.provider.Download(f.ctx, tmpFs); err != nil {
		return fmt.Errorf("cannot download to tmp directory: %w", err)
	}

	targetDirName, err := f.getTargetDirName(tmpFs.Fs)
	if err != nil {
		return fmt.Errorf("cannot get a target dir name: %w", err)
	}

	if targetDirName == currentTargetDir {
		return nil
	}

	if currentTargetDir != "" {
		if err := fs.RemoveAll(currentTargetDir); err != nil {
			return fmt.Errorf("cannot remove current target dir: %w", err)
		}
	}

	if err := fs.Rename(tmpDir, targetDirName); err != nil {
		return fmt.Errorf("cannot rename tmp dir to target one: %w", err)
	}

	if err := f.provider.Open(f.getTargetFs(targetDirName)); err != nil {
		return fmt.Errorf("cannot open a target dir: %w", err)
	}

	return nil
}

func (f *fsUpdater) getTargetDir() (string, error) {
	baseFs := f.getBaseFs()

	infos, err := baseFs.ReadDir(".")
	if err != nil {
		return "", fmt.Errorf("cannot read base directory: %w", err)
	}

	for _, v := range infos {
		if v.IsDir() && strings.HasPrefix(v.Name(), FsTargetDirPrefix) {
			return v.Name(), nil
		}
	}

	return "", errNoTargetDir
}

func (f *fsUpdater) getBaseFs() afero.Afero {
	return afero.Afero{
		Fs: afero.NewBasePathFs(afero.NewOsFs(), f.provider.BaseDirectory()),
	}
}

func (f *fsUpdater) getTargetFs(name string) afero.Fs {
	return afero.NewBasePathFs(f.getBaseFs().Fs, name)
}

func (f *fsUpdater) getTargetDirName(fs afero.Fs) (string, error) {
	hasher := sha256.New()
	startSign := []byte{0}
	fileSign := []byte{1}

	err := afero.Walk(fs, ".", func(path string, info os.FileInfo, err error) error {
		hasher.Write(startSign)      // nolint: errcheck
		io.WriteString(hasher, path) // nolint: errcheck

		switch {
		case err != nil:
			return err
		case info.IsDir():
			return nil
		}

		fp, err := fs.Open(path)
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

	return FsTargetDirPrefix + hex.EncodeToString(hasher.Sum(nil)), nil
}
