package topolib

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type FsUpdaterTestSuite struct {
	suite.Suite

	ctxCancel    context.CancelFunc
	providerMock *OfflineProviderMock
	loggerMock   *LoggerMock
	u            *fsUpdater
	baseDir      string
}

func (suite *FsUpdaterTestSuite) SetupTest() {
	baseDir, err := ioutil.TempDir("", "fs_updater_test_suite_")
	suite.baseDir = baseDir

	suite.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	suite.ctxCancel = cancel
	suite.providerMock = &OfflineProviderMock{}
	suite.loggerMock = &LoggerMock{}
	suite.u = &fsUpdater{
		OfflineProvider: suite.providerMock,
		ctx:             ctx,
		ctxCancel:       cancel,
		logger:          suite.loggerMock,
		fs:              fsDir{Dir: baseDir},
		stats:           &UsageStats{},
	}

	suite.providerMock.On("Shutdown")
	suite.providerMock.On("Name").Return("providerMock").Maybe()
	suite.providerMock.On("BaseDirectory").Return(baseDir).Maybe()
	suite.providerMock.On("UpdateEvery").Return(time.Minute).Maybe()
	suite.loggerMock.On("UpdateError", mock.Anything, mock.Anything).Maybe()
	suite.loggerMock.On("UpdateInfo", mock.Anything, mock.Anything).Maybe()
}

func (suite *FsUpdaterTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.u.Shutdown()
	suite.providerMock.AssertExpectations(suite.T())
	suite.loggerMock.AssertExpectations(suite.T())

	os.RemoveAll(suite.baseDir)
}

func (suite *FsUpdaterTestSuite) TestInitialCleaning() {
	targetDir, err := ioutil.TempDir(suite.baseDir, FsTargetDirPrefix)

	suite.NoError(err)

	_, err = ioutil.TempFile(suite.baseDir, "")

	suite.NoError(err)

	_, err = ioutil.TempDir(suite.baseDir, "")

	suite.NoError(err)

	errToCheck := errors.New("new error")

	suite.providerMock.On("Open", targetDir).Return(errToCheck)
	suite.providerMock.On("Download", mock.Anything, mock.Anything).Return(errToCheck)

	suite.Error(suite.u.Start())

	infos, err := ioutil.ReadDir(suite.baseDir)

	suite.NoError(err)
	suite.Len(infos, 0)
}

func (suite *FsUpdaterTestSuite) TestOk() {
	targetDir, err := ioutil.TempDir(suite.baseDir, FsTargetDirPrefix)

	suite.NoError(err)

	defer os.RemoveAll(targetDir)

	suite.providerMock.On("Download", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		tmpDir := args.String(1)

		suite.NoError(
			ioutil.WriteFile(filepath.Join(tmpDir, "filename"),
				[]byte("Hello"),
				0644))
		suite.NoError(os.MkdirAll(filepath.Join(tmpDir, "path", "to"), 0777))
		suite.NoError(ioutil.WriteFile(filepath.Join(tmpDir, "path", "to", "file"),
			[]byte("OK"),
			0644))
	})
	suite.providerMock.On("Open", mock.Anything).Return(nil)

	suite.NoError(suite.u.Start())

	time.Sleep(200 * time.Millisecond)

	infos, err := ioutil.ReadDir(suite.baseDir)

	suite.NoError(err)
	suite.Len(infos, 1)
	suite.Equal(
		"target_2a58c90eb7ef89d2aaddcfbe3b46ef2bea493915b54c2e4906a03e698be49dc3",
		infos[0].Name())
}

func TestFsUpdater(t *testing.T) {
	suite.Run(t, &FsUpdaterTestSuite{})
}
