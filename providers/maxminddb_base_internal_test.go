package providers

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type MaxMindDBBaseTestSuite struct {
	suite.Suite

	m *maxmindBase
}

func (suite *MaxMindDBBaseTestSuite) SetupTest() {
	suite.m = &maxmindBase{}
}

func (suite *MaxMindDBBaseTestSuite) TearDownTest() {
    suite.m.Shutdown()
}

func (suite *MaxMindDBBaseTestSuite) TestOpenErrorNoFile() {
	fs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()),
		filepath.Join("testdata", "maxmind", "error"))

	suite.Error(suite.m.Open(fs.(*afero.BasePathFs)))
}

func (suite *MaxMindDBBaseTestSuite) TestOpenErrorBadFile() {
	fs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()),
		filepath.Join("testdata", "maxmind", "error", "target_xxx"))

	suite.Error(suite.m.Open(fs.(*afero.BasePathFs)))
}

func (suite *MaxMindDBBaseTestSuite) TestOpenOk() {
	fs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()),
		filepath.Join("testdata", "maxmind", "ok", "target_xxx"))

	suite.NoError(suite.m.Open(fs.(*afero.BasePathFs)))
    suite.NotNil(suite.m.dbReader)
}

func (suite *MaxMindDBBaseTestSuite) TestLookupNotReady() {
    _, err := suite.m.Lookup(context.Background(), net.ParseIP("81.2.69.142"))

    suite.True(errors.Is(err, ErrDatabaseIsNotReadyYet))
}

func (suite *MaxMindDBBaseTestSuite) TestLookupBadIP() {
	fs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()),
		filepath.Join("testdata", "maxmind", "ok", "target_xxx"))

	suite.m.Open(fs.(*afero.BasePathFs))

    _, err := suite.m.Lookup(context.Background(), nil)

    suite.Error(err)
}

func (suite *MaxMindDBBaseTestSuite) TestLookupOk() {
	fs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()),
		filepath.Join("testdata", "maxmind", "ok", "target_xxx"))

	suite.m.Open(fs.(*afero.BasePathFs))

    result, err := suite.m.Lookup(context.Background(), net.ParseIP("81.2.69.142"))

    suite.NoError(err)
    suite.Equal("GB", result.CountryCode)
    suite.Equal("London", result.City)
}

func TestMaxMindDBBase(t *testing.T) {
	suite.Run(t, &MaxMindDBBaseTestSuite{})
}
