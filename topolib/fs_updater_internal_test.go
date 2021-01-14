package topolib

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
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
	ctx, cancel := context.WithCancel(context.Background())
	suite.ctxCancel = cancel
	suite.providerMock = &OfflineProviderMock{}
	suite.loggerMock = &LoggerMock{}
	suite.u = &fsUpdater{
		ctx:      ctx,
		cancel:   cancel,
		logger:   suite.loggerMock,
		provider: suite.providerMock,
	}

	baseDir, err := ioutil.TempDir("", "fs_updater_test_suite_")
	suite.baseDir = baseDir

	suite.NoError(err)

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

func (suite *FsUpdaterTestSuite) TestName() {
	suite.Equal("providerMock", suite.u.Name())
}

func (suite *FsUpdaterTestSuite) TestLookup() {
	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1").To4()
	res := ProviderLookupResult{}

	suite.providerMock.On("Lookup", ctx, ip).Return(res, nil)

	r, err := suite.u.Lookup(ctx, ip)

	suite.NoError(err)
	suite.Equal(res, r)
}

func (suite *FsUpdaterTestSuite) TestInitialCleaning() {
	targetDir, err := ioutil.TempDir(suite.baseDir, FsTargetDirPrefix)

	suite.NoError(err)

	_, err = ioutil.TempFile(suite.baseDir, "")

	suite.NoError(err)

	_, err = ioutil.TempDir(suite.baseDir, "")

	suite.NoError(err)

	errToCheck := errors.New("new error")

	suite.providerMock.On("Open", mock.Anything).Return(errToCheck).Run(func(args mock.Arguments) {
		fp, err := args.Get(0).(afero.Fs).Create("myfile")

		suite.NoError(err)

		fp.WriteString("hello")

		content, err := ioutil.ReadFile(filepath.Join(targetDir, "myfile"))

		suite.NoError(err)

		suite.Equal("hello", string(content))
	})

	err = suite.u.Start()

	suite.True(errors.Is(err, errToCheck))

	infos, err := ioutil.ReadDir(suite.baseDir)

	suite.NoError(err)
	suite.Len(infos, 1)
	suite.Equal(filepath.Base(targetDir), infos[0].Name())
}

func (suite *FsUpdaterTestSuite) TestOk() {
	targetDir, err := ioutil.TempDir(suite.baseDir, FsTargetDirPrefix)

	suite.NoError(err)

	defer os.RemoveAll(targetDir)

	suite.providerMock.On("Download", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fs := args.Get(1).(afero.Fs)

		fp, err := fs.Create("filename")

		suite.NoError(err)

		fp.WriteString("Hello")

		suite.NoError(fs.MkdirAll(filepath.Join("path", "to"), 0777))

		fp, err = fs.Create(filepath.Join("path", "to", "file"))

		suite.NoError(err)

		fp.WriteString("OK")
	})
	suite.providerMock.On("Open", mock.Anything).Return(nil)

	suite.NoError(suite.u.Start())

	time.Sleep(200 * time.Millisecond)

	infos, err := ioutil.ReadDir(suite.baseDir)

	suite.NoError(err)
	suite.Len(infos, 1)
	suite.Equal(
		"target_4c182b523da15532e3097f3a763615925df2e961939ff4930f5945dfd953c714",
		infos[0].Name())
}

func TestFsUpdater(t *testing.T) {
	suite.Run(t, &FsUpdaterTestSuite{})
}
