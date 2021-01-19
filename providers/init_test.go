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

type OnlineProviderTestSuite struct {
	ProviderTestSuite

	prov topolib.Provider
}

type HTTPMockMixin struct{}

func (suite *HTTPMockMixin) SetupSuite() {
	httpmock.Activate()
}

func (suite *HTTPMockMixin) TearDownSuite() {
	httpmock.DeactivateAndReset()
}

func (suite *HTTPMockMixin) TearDownTest() {
	httpmock.Reset()
}
