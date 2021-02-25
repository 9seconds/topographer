package topolib_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/suite"
)

type HTTPClientTestSuite struct {
	suite.Suite

	httpbinEndpoint *httptest.Server
	c               topolib.HTTPClient
}

func (suite *HTTPClientTestSuite) SetupSuite() {
	suite.httpbinEndpoint = httptest.NewServer(httpbin.NewHTTPBin().Handler())
}

func (suite *HTTPClientTestSuite) TearDownSuite() {
	suite.httpbinEndpoint.Close()
}

func (suite *HTTPClientTestSuite) SetupTest() {
	suite.c = topolib.NewHTTPClient(suite.httpbinEndpoint.Client(),
		"test",
		100*time.Millisecond,
		1,
		5,
		time.Minute,
		time.Minute)
}

func (suite *HTTPClientTestSuite) TestRateLimiter() {
	now := time.Now()
	wg := &sync.WaitGroup{}

	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()

			req, _ := http.NewRequest("GET", suite.httpbinEndpoint.URL+"/get", nil)
			resp, err := suite.c.Do(req)

			suite.NoError(err)
			suite.Equal(http.StatusOK, resp.StatusCode)
		}()
	}

	wg.Wait()

	suite.True(time.Since(now) > 700*time.Millisecond)
	suite.WithinDuration(now, time.Now(), 12*100*time.Millisecond)
}

func (suite *HTTPClientTestSuite) TestBadStatus() {
	req, _ := http.NewRequest("GET", suite.httpbinEndpoint.URL+"/status/500", nil)
	_, err := suite.c.Do(req)

	suite.Error(err)
}

func (suite *HTTPClientTestSuite) TestCannotDial() {
	req, _ := http.NewRequest("GET", suite.httpbinEndpoint.URL+"1"+"/status/500", nil)
	_, err := suite.c.Do(req)

	suite.Error(err)
}

func TestHTTPClient(t *testing.T) {
	suite.Run(t, &HTTPClientTestSuite{})
}
