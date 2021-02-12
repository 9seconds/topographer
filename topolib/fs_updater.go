package topolib

import (
	"context"
	"fmt"
	"os"
	"time"
)

type fsUpdater struct {
	OfflineProvider

	ctx       context.Context
	ctxCancel context.CancelFunc
	logger    Logger
	fs        fsDir
	stats     *UsageStats
}

func (f *fsUpdater) Start() error {
	targetDir, _, err := f.fs.GetTargetDir()
	if err != nil {
		return fmt.Errorf("cannot get target dir: %w", err)
	}

	if err := f.fs.Cleanup(targetDir); err != nil {
		return fmt.Errorf("cannot do startup cleanup: %w", err)
	}

	if targetDir != "" {
		if err := f.Open(targetDir); err == nil {
			go f.runBgUpdate()

			return nil
		}
	}

	if err := f.fs.Cleanup(); err != nil {
		return fmt.Errorf("cannot make full cleanup: %w", err)
	}

	if err := f.doUpdate(); err != nil {
		return fmt.Errorf("cannot fetch databases: %w", err)
	}

	f.logger.UpdateInfo(f.Name())

	go f.runBgUpdate()

	return nil
}

func (f *fsUpdater) Shutdown() {
	f.ctxCancel()
	f.OfflineProvider.Shutdown()
}

func (f *fsUpdater) runBgUpdate() {
	_, modTime, _ := f.fs.GetTargetDir()
	duration := time.Since(modTime)

	if duration > 0 {
		timer := time.NewTimer(duration)
		defer func() {
			timer.Stop()

			select {
			case <-timer.C:
			default:
			}
		}()

		select {
		case <-f.ctx.Done():
			return
		case <-timer.C:
		}
	}

	if err := f.doUpdate(); err != nil {
		f.logger.UpdateError(f.Name(), err)
	} else {
		f.logger.UpdateInfo(f.Name())
	}

	ticker := time.NewTicker(f.UpdateEvery())
	defer func() {
		ticker.Stop()

		select {
		case <-ticker.C:
		default:
		}
	}()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			if err := f.doUpdate(); err != nil {
				f.logger.UpdateError(f.Name(), err)
			} else {
				f.logger.UpdateInfo(f.Name())
			}
		}
	}
}

func (f *fsUpdater) doUpdate() error {
	tmpDir, err := f.fs.TempDir()
	if err != nil {
		return fmt.Errorf("cannot make a temporary dir: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	if err := f.Download(f.ctx, tmpDir); err != nil {
		return fmt.Errorf("cannot download databases: %w", err)
	}

	newTargetDir, needToReopen, err := f.fs.Promote(tmpDir)
	if err != nil {
		return fmt.Errorf("cannot promote tmp dir: %w", err)
	}

	if needToReopen {
		if err := f.Open(newTargetDir); err != nil {
			os.RemoveAll(newTargetDir)

			return fmt.Errorf("cannot open a new target dir: %w", err)
		}
	}

	f.fs.Cleanup(newTargetDir) // nolint: errcheck
	f.stats.notifyUpdated()

	return nil
}

func newFsUpdater(provider OfflineProvider, logger Logger, stats *UsageStats) (OfflineProvider, error) {
	ctx, cancel := context.WithCancel(context.Background())

	updater := &fsUpdater{
		OfflineProvider: provider,
		ctx:             ctx,
		ctxCancel:       cancel,
		logger:          logger,
		fs:              fsDir{Dir: provider.BaseDirectory()},
		stats:           stats,
	}

	if err := updater.Start(); err != nil {
		return nil, fmt.Errorf("cannot start fs updater for provider %s: %w",
			provider.Name(),
			err)
	}

	return updater, nil
}
