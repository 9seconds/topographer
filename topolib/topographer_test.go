package topolib_test

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TopographerTestSuite struct {
	suite.Suite

	t                    *topolib.Topographer
	tmpDir               string
	providerMocks        []*ProviderMock
	offlineProviderMocks []*OfflineProviderMock
	logMock              *LoggerMock
}

func (suite *TopographerTestSuite) SetupTest() {
	suite.logMock = &LoggerMock{}
	suite.providerMocks = []*ProviderMock{{}, {}}
	suite.offlineProviderMocks = []*OfflineProviderMock{{}}
	suite.tmpDir, _ = ioutil.TempDir("", "topo_test_")

	suite.logMock.On("UpdateInfo", mock.Anything, mock.Anything).Maybe()
	suite.logMock.On("LookupError", mock.Anything, mock.Anything, mock.Anything).Maybe()
	suite.logMock.On("UpdateError", mock.Anything, mock.Anything).Maybe()

	providers := []topolib.Provider{}

	for idx, v := range suite.providerMocks {
		v.On("Name").Return("p" + strconv.Itoa(idx)).Maybe()

		providers = append(providers, v)
	}

	for idx, v := range suite.offlineProviderMocks {
		v.On("Shutdown").Once()
		v.On("BaseDirectory").Return(suite.tmpDir).Maybe()
		v.On("UpdateEvery").Return(time.Minute).Maybe()
		v.On("Download", mock.Anything, mock.Anything).Return(nil).Maybe()
		v.On("Open", mock.Anything).Return(nil).Maybe()
		v.On("Name").Return("o" + strconv.Itoa(idx)).Maybe()

		providers = append(providers, v)
	}

	suite.t, _ = topolib.NewTopographer(providers, suite.logMock, 10)
}

func (suite *TopographerTestSuite) TearDownTest() {
	suite.t.Shutdown()

	suite.logMock.AssertExpectations(suite.T())

	for _, v := range suite.providerMocks {
		v.AssertExpectations(suite.T())
	}

	for _, v := range suite.offlineProviderMocks {
		v.AssertExpectations(suite.T())
	}

	time.Sleep(100 * time.Millisecond)

	os.RemoveAll(suite.tmpDir)
}

func (suite *TopographerTestSuite) TestResolveShutdown() {
	suite.t.Shutdown()

	ip := net.ParseIP("127.0.0.1")
	res, err := suite.t.Resolve(context.Background(), ip, nil)

	suite.True(errors.Is(err, topolib.ErrTopographerShutdown))
	suite.False(res.OK())
}

func (suite *TopographerTestSuite) TestResolveGivenCtxClosed() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	ip := net.ParseIP("127.0.0.1")
	res, err := suite.t.Resolve(ctx, ip, nil)

	suite.NoError(err)
	suite.False(res.OK())
}

func (suite *TopographerTestSuite) TestResolveUnknownProvider() {
	ip := net.ParseIP("127.0.0.1")
	res, err := suite.t.Resolve(context.Background(), ip, []string{"o0", "u"})

	suite.EqualError(err, "provider u is unknown")
	suite.False(res.OK())
}

func (suite *TopographerTestSuite) TestResolveAllShutdown() {
	suite.t.Shutdown()

	ip := net.ParseIP("127.0.0.1")
	res, err := suite.t.ResolveAll(context.Background(), []net.IP{ip}, nil)

	suite.True(errors.Is(err, topolib.ErrTopographerShutdown))
	suite.Empty(res)
}

func (suite *TopographerTestSuite) TestResolveAllGivenCtxClosed() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	ip := net.ParseIP("127.0.0.1")
	res, err := suite.t.ResolveAll(ctx, []net.IP{ip}, nil)

	suite.NoError(err)
	suite.Empty(res)
}

func TestTopographer(t *testing.T) {
	suite.Run(t, &TopographerTestSuite{})
}
