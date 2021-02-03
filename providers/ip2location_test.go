package providers_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/stretchr/testify/suite"
)

const ip2locationEnvApiKey = "IP2LOCATION_API_KEY"

type MockedIP2LocationTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedIP2LocationTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	prov, err := providers.NewIP2Location(suite.http,
		time.Minute,
		suite.tmpDir,
		"token",
		"DBTEST")
	if err != nil {
		panic(err)
	}

	suite.prov = prov
}

func (suite *MockedIP2LocationTestSuite) TearDownTest() {
	suite.HTTPMockMixin.TearDownTest()
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

type IntegrationIP2LocationTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationIP2LocationTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	prov, err := providers.NewIP2Location(suite.http,
		time.Minute,
		"",
		os.Getenv(ip2locationEnvApiKey),
		"")

	suite.NoError(err)

	suite.prov = prov
}

func (suite *IntegrationIP2LocationTestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.SetupTest()
	suite.TmpDirTestSuite.SetupTest()
}

func (suite *IntegrationIP2LocationTestSuite) TestFull() {
	ctx := context.Background()

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))
}

func TestIP2Location(t *testing.T) {
	suite.Run(t, &MockedIP2LocationTestSuite{})
}

func TestIntegrationIP2Location(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
		return
	}

	if os.Getenv(ip2locationEnvApiKey) == "" {
		t.Skip("Skipped because " + ip2locationEnvApiKey + " environment variable is empty")
		return
	}

	suite.Run(t, &IntegrationIP2LocationTestSuite{})
}
