package providers

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MaxmindDBBaseTestSuite struct {
	suite.Suite

	m      *maxmindBase
	tmpDir string
}

func (suite *MaxmindDBBaseTestSuite) SetupTest() {
	dir, err := ioutil.TempDir("", "test_")
	if err != nil {
		panic(err)
	}

	suite.tmpDir = dir
	suite.m = &maxmindBase{}
}

func (suite *MaxmindDBBaseTestSuite) TearDownTest() {
	suite.m.Shutdown()
	os.Chmod(suite.tmpDir, 0777) // nolint: errcheck
	os.RemoveAll(suite.tmpDir)
}

func (suite *MaxmindDBBaseTestSuite) GetTestdataPath() string {
	absPath, err := filepath.Abs("testdata")
	if err != nil {
		panic(err)
	}

	return absPath
}

func (suite *MaxmindDBBaseTestSuite) TestOpenErrorNoFile() {
	if err := os.Chmod(suite.tmpDir, 0400); err != nil {
		panic(err)
	}

	suite.Error(suite.m.Open(suite.tmpDir))
}

func (suite *MaxmindDBBaseTestSuite) TestOpenErrorBadFile() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "error", "target_xxx")

	suite.Error(suite.m.Open(path))
}

func (suite *MaxmindDBBaseTestSuite) TestOpenOk() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "ok", "target_xxx")

	suite.NoError(suite.m.Open(path))
	suite.NotNil(suite.m.dbReader)
}

func (suite *MaxmindDBBaseTestSuite) TestLookupNotReady() {
	_, err := suite.m.Lookup(context.Background(), net.ParseIP("81.2.69.142"))

	suite.True(errors.Is(err, ErrDatabaseIsNotReadyYet))
}

func (suite *MaxmindDBBaseTestSuite) TestLookupBadIP() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "ok", "target_xxx")

	suite.m.Open(path) // nolint: errcheck

	_, err := suite.m.Lookup(context.Background(), nil)

	suite.Error(err)
}

func (suite *MaxmindDBBaseTestSuite) TestLookupOk() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "ok", "target_xxx")

	suite.m.Open(path) // nolint: errcheck

	result, err := suite.m.Lookup(context.Background(), net.ParseIP("81.2.69.142"))

	suite.NoError(err)
	suite.Equal("GB", result.CountryCode.String())
	suite.Equal("London", result.City)
}

func TestMaxmindDBBase(t *testing.T) {
	suite.Run(t, &MaxmindDBBaseTestSuite{})
}
