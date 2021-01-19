package providers_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type ProviderTestSuite struct {
	suite.Suite

	http          topolib.HTTPClient
	baseDirectory string
}

func (suite *ProviderTestSuite) SetupTest() {
	suite.http = topolib.NewHTTPClient(&http.Client{},
		"test-agent",
		time.Millisecond,
		100)

	dir, err := ioutil.TempDir("", "topo_int_test_")
	if err != nil {
		panic(err)
	}

	suite.baseDirectory = dir
}

func (suite *ProviderTestSuite) TearDownTest() {
	os.RemoveAll(suite.baseDirectory)
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

func (suite *MockedProviderTestSuite) TearDownTest() {
	httpmock.Reset()
}
