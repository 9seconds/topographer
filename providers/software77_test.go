package providers_test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/stretchr/testify/suite"
)

type MockedSoftware77TestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedSoftware77TestSuite) BaseDirectory() string {
	return filepath.Join(suite.GetTestdataPath(), "software77")
}

func (suite *MockedSoftware77TestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewSoftware77(suite.http,
		time.Minute,
		suite.BaseDirectory())
}

func (suite *MockedSoftware77TestSuite) TearDownTest() {
	suite.HTTPMockMixin.TearDownTest()
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

type IntegrationSoftware77TestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationSoftware77TestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewSoftware77(suite.http,
		time.Minute,
		"")
}

func (suite *IntegrationSoftware77TestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *IntegrationSoftware77TestSuite) TestFull() {
	ctx := context.Background()

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))
	suite.NoError(suite.prov.Open(suite.tmpDir))
	suite.NoError(suite.prov.Open(suite.tmpDir))

	res, err := suite.prov.Lookup(ctx, net.ParseIP("80.80.80.80"))

	suite.NoError(err)
	suite.Equal("NL", res.CountryCode)
}

func TestSoftware77(t *testing.T) {
	suite.Run(t, &MockedSoftware77TestSuite{})
}

func TestIntegrationSoftware77(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
	}

	suite.Run(t, &IntegrationSoftware77TestSuite{})
}
