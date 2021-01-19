package providers_test

import (
	"net/http"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type ProviderTestSuite struct {
	suite.Suite

	http topolib.HTTPClient
}

func (suite *ProviderTestSuite) SetupTest() {
	suite.http = topolib.NewHTTPClient(&http.Client{},
		"test-agent",
		time.Millisecond,
		100)
}

type MockedProviderTestSuite struct {
	ProviderTestSuite
}

func (suite *MockedProviderTestSuite) SetupSuite() {
	httpmock.Activate()
}

func (suite *MockedProviderTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
}

func (suite *ProviderTestSuite) TearDownTest() {
	httpmock.Reset()
}
