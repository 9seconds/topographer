package topolib

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type PoolFuncMock struct {
	mock.Mock
}

func (m *PoolFuncMock) Do(arg interface{}) {
	m.Called(arg)
}

type PoolGroupRequestTestSuite struct {
	suite.Suite

	ctx           context.Context
	cancel        context.CancelFunc
	resultChannel chan ResolveResult
	pool          *ants.PoolWithFunc
	wg            *sync.WaitGroup
	poolFunc      *PoolFuncMock
	providerMocks []*ProviderMock
	pgr           *poolGroupRequest
}

func (suite *PoolGroupRequestTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.resultChannel = make(chan ResolveResult, 1)
	suite.wg = &sync.WaitGroup{}
	suite.providerMocks = []*ProviderMock{{}, {}}
	suite.poolFunc = &PoolFuncMock{}
	suite.pool, _ = ants.NewPoolWithFunc(5, suite.poolFunc.Do)

	providers := []Provider{}

	for _, v := range suite.providerMocks {
		providers = append(providers, v)
	}

	suite.pgr = newPoolGroupRequest(suite.ctx,
		suite.resultChannel,
		providers,
		suite.wg,
		suite.pool)
}

func (suite *PoolGroupRequestTestSuite) TearDownTest() {
	suite.poolFunc.AssertExpectations(suite.T())

	for _, v := range suite.providerMocks {
		v.AssertExpectations(suite.T())
	}

	suite.wg.Wait()
	suite.cancel()
}

func (suite *PoolGroupRequestTestSuite) TestParentClosed() {
	suite.cancel()

	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1")

	suite.True(errors.Is(suite.pgr.Do(ctx, ip), ErrContextIsClosed))
	suite.True(errors.Is(suite.pgr.Do(ctx, ip), ErrContextIsClosed))
}

func (suite *PoolGroupRequestTestSuite) TestSelfClosed() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ip := net.ParseIP("127.0.0.1")

	suite.True(errors.Is(suite.pgr.Do(ctx, ip), ErrContextIsClosed))
	suite.True(errors.Is(suite.pgr.Do(ctx, ip), ErrContextIsClosed))
}

func (suite *PoolGroupRequestTestSuite) TestHappyPath() {
	ip := net.ParseIP("127.0.0.1")

	suite.poolFunc.On("Do", mock.Anything).Once().Run(func(args mock.Arguments) {
		req := args.Get(0).(*resolveIPRequest)

		req.wg.Done()
		req.resultChannel <- ResolveResult{City: "Moscow"}
	})

	suite.NoError(suite.pgr.Do(context.Background(), ip))

	r := <-suite.resultChannel

	suite.Equal("Moscow", r.City)
}

func TestPoolGroupRequest(t *testing.T) {
	suite.Run(t, &PoolGroupRequestTestSuite{})
}
