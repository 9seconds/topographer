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

type MockedKeyCDNTestSuite struct {
	MockedProviderTestSuite

	prov topolib.Provider
}

func (suite *MockedKeyCDNTestSuite) SetupTest() {
	suite.MockedProviderTestSuite.SetupTest()

	suite.prov = providers.NewKeyCDN(suite.http)
}

func (suite *MockedKeyCDNTestSuite) TestName() {
	suite.Equal(providers.NameKeyCDN, suite.prov.Name())
}

func (suite *MockedKeyCDNTestSuite) TestLookupClosedContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	_, err := suite.prov.Lookup(ctx, net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedKeyCDNTestSuite) TestLookupFailed() {
	httpmock.RegisterResponder("GET",
		"https://tools.keycdn.com/geo.json?host=23.22.13.113",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedKeyCDNTestSuite) TestLookupBadJSON() {
	httpmock.RegisterResponder("GET",
		"https://tools.keycdn.com/geo.json?host=23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, `{[`))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Error(err)
}

func (suite *MockedKeyCDNTestSuite) TestOk() {
	httpmock.RegisterResponder("GET",
		"https://tools.keycdn.com/geo.json?host=23.22.13.113",
		httpmock.NewStringResponder(http.StatusOK, `{
  "status":"success",
  "description":"Data successfully received.",
  "data":{
    "geo":{
      "host":"23.22.13.113",
      "ip":"23.22.13.113",
      "rdns":"ec2-23-22-13-113.compute-1.amazonaws.com",
      "asn":14618,
      "isp":"AMAZON-AES",
      "country_name":"United States",
      "country_code":"US",
      "region_name":"Virginia",
      "region_code":"VA",
      "city":"Ashburn",
      "postal_code":"20149",
      "continent_name":"North America",
      "continent_code":"NA",
      "latitude":39.0481,
      "longitude":-77.4728,
      "metro_code":511,
      "timezone":"America\/New_York",
      "datetime":"2021-01-19 02:38:56"
    }
  }
}`))

	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("US", strings.ToUpper(result.CountryCode))
	suite.NoError(err)
}

type IntegrationKeyCDNTestSuite struct {
	ProviderTestSuite

	prov topolib.Provider
}

func (suite *IntegrationKeyCDNTestSuite) SetupTest() {
	suite.ProviderTestSuite.SetupTest()

	suite.prov = providers.NewKeyCDN(suite.http)
}

func (suite *IntegrationKeyCDNTestSuite) TestLookup() {
	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("US", strings.ToUpper(result.CountryCode))
	suite.NoError(err)
}

func TestKeyCDN(t *testing.T) {
	suite.Run(t, &MockedKeyCDNTestSuite{})
}

func TestIntegrationKeyCDN(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
		return
	}

	suite.Run(t, &IntegrationKeyCDNTestSuite{})
}
