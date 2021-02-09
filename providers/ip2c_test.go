package providers_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/9seconds/topographer/providers"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type MockedIP2CTestSuite struct {
	OnlineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedIP2CTestSuite) SetupTest() {
	suite.OnlineProviderTestSuite.SetupTest()

	suite.prov = providers.NewIP2C(suite.http)
}

func (suite *MockedIP2CTestSuite) TestName() {
	suite.Equal(providers.NameIP2C, suite.prov.Name())
}

func (suite *MockedIP2CTestSuite) TestLookupClosedContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	_, err := suite.prov.Lookup(ctx, net.ParseIP("5.6.7.8"))

	suite.Error(err)
}

func (suite *MockedIP2CTestSuite) TestLookupFailed() {
	httpmock.RegisterResponder("GET",
		"https://ip2c.org?dec=84281096",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("5.6.7.8"))

	suite.Error(err)
}

func (suite *MockedIP2CTestSuite) TestLookupIncorrectFormat() {
	httpmock.RegisterResponder("GET",
		"https://ip2c.org?dec=84281096",
		httpmock.NewStringResponder(http.StatusOK, "1;DE"))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("5.6.7.8"))

	suite.Error(err)
}

func (suite *MockedIP2CTestSuite) TestLookupIncorrectStatusCode() {
	httpmock.RegisterResponder("GET",
		"https://ip2c.org?dec=84281096",
		httpmock.NewStringResponder(http.StatusOK, "0;DE;DEU;Germany"))

	_, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("5.6.7.8"))

	suite.Error(err)
}

func (suite *MockedIP2CTestSuite) TestLookupOk() {
	httpmock.RegisterResponder("GET",
		"https://ip2c.org?dec=84281096",
		httpmock.NewStringResponder(http.StatusOK, "1;DE;DEU;Germany"))

	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("5.6.7.8"))

	suite.NoError(err)
	suite.Equal("DE", strings.ToUpper(result.CountryCode))
}

type IntegrationIP2CTestSuite struct {
	OnlineProviderTestSuite
}

func (suite *IntegrationIP2CTestSuite) SetupTest() {
	suite.OnlineProviderTestSuite.SetupTest()

	suite.prov = providers.NewIP2C(suite.http)
}

func (suite *IntegrationIP2CTestSuite) TestLookup() {
	result, err := suite.prov.Lookup(context.Background(),
		net.ParseIP("23.22.13.113"))

	suite.Equal("US", strings.ToUpper(result.CountryCode))
	suite.NoError(err)
}

func TestIP2C(t *testing.T) {
	suite.Run(t, &MockedIP2CTestSuite{})
}

func TestIntegrationIP2C(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
	}

	suite.Run(t, &IntegrationIP2CTestSuite{})
}
