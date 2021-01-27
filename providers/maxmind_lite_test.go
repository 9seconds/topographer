package providers_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/stretchr/testify/suite"
)

const envApiKey = "MAXMIND_API_KEY"

type MaxmindLiteTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

type IntegrationMaxmindLiteTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationMaxmindLiteTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()
}

func (suite *IntegrationMaxmindLiteTestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *IntegrationMaxmindLiteTestSuite) TestFull() {
	prov := providers.NewMaxmindLite(suite.http, time.Minute, "", map[string]string{
		"license_key": os.Getenv(envApiKey),
	})

	suite.NoError(prov.Download(context.Background(), suite.tmpDir))
	suite.NoError(prov.Open(suite.tmpDir))

	_, err := prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

	suite.NoError(err)
}

func TestMaxmindLite(t *testing.T) {
	suite.Run(t, &MaxmindLiteTestSuite{})
}

func TestIntegrationMaxmindLite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
		return
	}

	if os.Getenv(envApiKey) == "" {
		t.Skip("Skipped because " + envApiKey + " environment variable is empty")
		return
	}

	suite.Run(t, &IntegrationMaxmindLiteTestSuite{})
}
