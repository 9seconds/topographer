package topolib_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CachingProviderBaseTestSuite struct {
	suite.Suite

	p                     topolib.Provider
	mockedProvider        *ProviderMock
	mockedOfflineProvider *OfflineProviderMock
}

func (suite *CachingProviderBaseTestSuite) SetupTest() {
	suite.mockedProvider = &ProviderMock{}
	suite.mockedOfflineProvider = &OfflineProviderMock{}
}

func (suite *CachingProviderBaseTestSuite) TearDownTest() {
	suite.mockedProvider.AssertExpectations(suite.T())
	suite.mockedOfflineProvider.AssertExpectations(suite.T())
}

func (suite *CachingProviderBaseTestSuite) TestLookup() {
	ctx := context.Background()
	ip := net.ParseIP("80.80.81.81")

	result1, err := suite.p.Lookup(ctx, ip)

	suite.NoError(err)

	// ristretto is eventually consistent
	time.Sleep(100 * time.Millisecond)

	result2, err := suite.p.Lookup(ctx, ip)

	suite.NoError(err)

	suite.Equal(result1.City, result2.City)
	suite.Equal(result1.CountryCode, result2.CountryCode)
}

type CachingProviderTestSuite struct {
	CachingProviderBaseTestSuite
}

func (suite *CachingProviderTestSuite) SetupTest() {
	suite.CachingProviderBaseTestSuite.SetupTest()

	suite.p = topolib.NewCachingProvider(suite.mockedProvider, 100, time.Minute)
	call := suite.mockedProvider.On("Lookup", mock.Anything, mock.Anything)

	call.Return(topolib.ProviderLookupResult{City: "Nizhny Novgorod", CountryCode: "RU"}, nil)
	call.Once()
}

type OfflineCachingProviderTestSuite struct {
	CachingProviderBaseTestSuite
}

func (suite *OfflineCachingProviderTestSuite) SetupTest() {
	suite.CachingProviderBaseTestSuite.SetupTest()

	suite.p = topolib.NewCachingOfflineProvider(suite.mockedOfflineProvider, 100, time.Minute)
	call := suite.mockedOfflineProvider.On("Lookup", mock.Anything, mock.Anything)

	call.Return(topolib.ProviderLookupResult{City: "Nizhny Novgorod", CountryCode: "RU"}, nil)
	call.Once()
}

func TestCachingProvider(t *testing.T) {
	suite.Run(t, &CachingProviderTestSuite{})
}

func TestOfflineCachingProvider(t *testing.T) {
	suite.Run(t, &OfflineCachingProviderTestSuite{})
}
