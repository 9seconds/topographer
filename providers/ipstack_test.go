package providers_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/9seconds/topographer/providers"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type MockedIPStackTestSuite struct {
	OnlineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedIPStackTestSuite) SetupTest() {
	suite.OnlineProviderTestSuite.SetupTest()

	prov, err := providers.NewIPStack(suite.http, "token", true)

	if err != nil {
		panic(err)
	}

	suite.prov = prov
}

func (suite *MockedIPStackTestSuite) TestName() {
	suite.Equal(providers.NameIPStack, suite.prov.Name())
}

func (suite *MockedIPStackTestSuite) TestLookupClosedContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	_, err := suite.prov.Lookup(ctx, net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPStackTestSuite) TestLookupFailed() {
	httpmock.RegisterResponder("GET",
		"https://api.ipstack.com/23.22.13.113",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPStackTestSuite) TestLookupBadJSON() {
	httpmock.RegisterResponder("GET",
		"https://api.ipstack.com/23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, "{["))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPStackTestSuite) TestLookupError() {
	httpmock.RegisterResponder("GET",
		"https://api.ipstack.com/23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, `{
"success": false,
"error": {"code": 105, "info": "", "type": "some_error"}
        }`))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPStackTestSuite) TestLookupOK() {
	httpmock.RegisterResponder("GET",
		"https://api.ipstack.com/23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, `{
"country_code": "RU", "city": "Moscow"
        }`))

	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("RU", result.CountryCode.String())
	suite.Equal("Moscow", result.City)
	suite.NoError(err)
}

type IntegrationIPStackTestSuite struct {
	OnlineProviderTestSuite
}

func (suite *IntegrationIPStackTestSuite) SetupTest() {
	suite.OnlineProviderTestSuite.SetupTest()

	prov, err := providers.NewIPStack(suite.http, os.Getenv("IPSTACK_API_KEY"), false)

	suite.NoError(err)

	suite.prov = prov
}

func (suite *IntegrationIPStackTestSuite) TestLookup() {
	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("US", result.CountryCode.String())
	suite.NoError(err)
}

func TestIPStack(t *testing.T) {
	suite.Run(t, &MockedIPStackTestSuite{})
}

func TestIntegrationIPStack(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped becaue of the short mode")
	}

	if os.Getenv("IPSTACK_API_KEY") == "" {
		t.Skip("Skipped because there is no IPSTACK_API_KEY in environment")
	}

	suite.Run(t, &IntegrationIPStackTestSuite{})
}
