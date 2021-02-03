package providers_test

import (
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

func TestIP2Location(t *testing.T) {
	suite.Run(t, &MockedIP2LocationTestSuite{})
}
