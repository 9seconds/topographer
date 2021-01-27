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

type MaxMindDBBaseTestSuite struct {
	suite.Suite

	m      *maxmindBase
	tmpDir string
}

func (suite *MaxMindDBBaseTestSuite) SetupTest() {
	dir, err := ioutil.TempDir("", "test_")
	if err != nil {
		panic(err)
	}

	suite.tmpDir = dir
	suite.m = &maxmindBase{}
}

func (suite *MaxMindDBBaseTestSuite) TearDownTest() {
	suite.m.Shutdown()
	os.Chmod(suite.tmpDir, 0777) // nolint: errcheck
	os.RemoveAll(suite.tmpDir)
}

func (suite *MaxMindDBBaseTestSuite) GetTestdataPath() string {
	absPath, err := filepath.Abs("testdata")
	if err != nil {
		panic(err)
	}

	return absPath
}

func (suite *MaxMindDBBaseTestSuite) TestOpenErrorNoFile() {
	if err := os.Chmod(suite.tmpDir, 0400); err != nil {
		panic(err)
	}

	suite.Error(suite.m.Open(suite.tmpDir))
}

func (suite *MaxMindDBBaseTestSuite) TestOpenErrorBadFile() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "error", "target_xxx")

	suite.Error(suite.m.Open(path))
}

func (suite *MaxMindDBBaseTestSuite) TestOpenOk() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "ok", "target_xxx")

	suite.NoError(suite.m.Open(path))
	suite.NotNil(suite.m.dbReader)
}

func (suite *MaxMindDBBaseTestSuite) TestLookupNotReady() {
	_, err := suite.m.Lookup(context.Background(), net.ParseIP("81.2.69.142"))

	suite.True(errors.Is(err, ErrDatabaseIsNotReadyYet))
}

func (suite *MaxMindDBBaseTestSuite) TestLookupBadIP() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "ok", "target_xxx")

	suite.m.Open(path) // nolint: errcheck

	_, err := suite.m.Lookup(context.Background(), nil)

	suite.Error(err)
}

func (suite *MaxMindDBBaseTestSuite) TestLookupOk() {
	path := filepath.Join(suite.GetTestdataPath(),
		"maxmind", "ok", "target_xxx")

	suite.m.Open(path) // nolint: errcheck

	result, err := suite.m.Lookup(context.Background(), net.ParseIP("81.2.69.142"))

	suite.NoError(err)
	suite.Equal("GB", result.CountryCode)
	suite.Equal("London", result.City)
}

func TestMaxMindDBBase(t *testing.T) {
	suite.Run(t, &MaxMindDBBaseTestSuite{})
}
