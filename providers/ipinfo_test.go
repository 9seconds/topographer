package providers_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/9seconds/topographer/providers"
	"github.com/9seconds/topographer/topolib"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type MockedIPInfoTestSuite struct {
	MockedProviderTestSuite

	prov topolib.Provider
}

func (suite *MockedIPInfoTestSuite) SetupTest() {
	suite.MockedProviderTestSuite.SetupTest()

	suite.prov = providers.NewIPInfo(suite.http, map[string]string{
		"auth_token": "token",
	})
}

func (suite *MockedIPInfoTestSuite) TestName() {
	suite.Equal(providers.NameIPInfo, suite.prov.Name())
}

func (suite *MockedIPInfoTestSuite) TestLookupClosedContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	_, err := suite.prov.Lookup(ctx, net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPInfoTestSuite) TestLookupFailed() {
	httpmock.RegisterResponder("GET",
		"https://ipinfo.io/23.22.13.113",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPInfoTestSuite) TestLookupBadJSON() {
	httpmock.RegisterResponder("GET",
		"https://ipinfo.io/23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, `{[`))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedIPInfoTestSuite) TestLookupOk() {
	httpmock.RegisterResponder("GET",
		"https://ipinfo.io/23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, `{
  "ip": "23.22.13.113",
  "hostname": "ec2-23-22-13-113.compute-1.amazonaws.com",
  "city": "Virginia Beach",
  "region": "Virginia",
  "country": "US",
  "loc": "36.7957,-76.0126",
  "org": "AS14618 Amazon.com, Inc.",
  "postal": "23479",
  "timezone": "America/New_York",
  "readme": "https://ipinfo.io/missingauth"
}`))

	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("US", strings.ToUpper(result.CountryCode))
	suite.NoError(err)
}

type IntegrationIPInfoTestSuite struct {
	ProviderTestSuite

	prov topolib.Provider
}

func (suite *IntegrationIPInfoTestSuite) SetupTest() {
	suite.ProviderTestSuite.SetupTest()

	suite.prov = providers.NewIPInfo(suite.http, map[string]string{})
}

func (suite *IntegrationIPInfoTestSuite) TestLookup() {
	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("US", strings.ToUpper(result.CountryCode))
	suite.NoError(err)
}

func TestIPInfo(t *testing.T) {
	suite.Run(t, &MockedIPInfoTestSuite{})
}

func TestIntegrationIPInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
		return
	}

	suite.Run(t, &IntegrationIPInfoTestSuite{})
}
