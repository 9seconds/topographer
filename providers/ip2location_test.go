package providers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

const ip2locationEnvApiKey = "IP2LOCATION_API_KEY"

type MockedIP2LocationTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedIP2LocationTestSuite) BaseDirectory() string {
	return filepath.Join(suite.GetTestdataPath(), "ip2location")
}

func (suite *MockedIP2LocationTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	prov, err := providers.NewIP2Location(suite.http,
		time.Minute,
		suite.BaseDirectory(),
		"token",
		"DBTEST")
	if err != nil {
		panic(err)
	}

	suite.prov = prov
}

func (suite *MockedIP2LocationTestSuite) TearDownTest() {
	suite.HTTPMockMixin.TearDownTest()
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *MockedIP2LocationTestSuite) TestName() {
	suite.Equal(providers.NameIP2Location, suite.prov.Name())
}

func (suite *MockedIP2LocationTestSuite) TestUpdateEvery() {
	suite.Equal(time.Minute, suite.prov.UpdateEvery())
}

func (suite *MockedIP2LocationTestSuite) TestBaseDirectory() {
	suite.Equal(suite.BaseDirectory(), suite.prov.BaseDirectory())
}

func (suite *MockedIP2LocationTestSuite) TestDownloadCancelledContext() {
    ctx, cancel := context.WithCancel(context.Background())

    cancel()

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedIP2LocationTestSuite) TestDownloadUrlNotFound() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://www.ip2location.com/download/",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedIP2LocationTestSuite) TestDownloadFSReadOnly() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://www.ip2location.com/download/",
		httpmock.NewStringResponder(http.StatusOK, "123"))

	suite.NoError(os.Chmod(suite.tmpDir, 0555))
	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedIP2LocationTestSuite) TestDownloadNoBin() {
	ctx := context.Background()

	zipBuf := &bytes.Buffer{}
	zipWriter := zip.NewWriter(zipBuf)

	fp, _ := zipWriter.Create("README.txt")

    fp.Write([]byte{1, 2, 3}) // nolint: errcheck

	zipWriter.Close()

	httpmock.RegisterResponder("GET",
		"https://www.ip2location.com/download/",
		httpmock.NewBytesResponder(http.StatusOK, zipBuf.Bytes()))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedIP2LocationTestSuite) TestDownloadOk() {
	ctx := context.Background()

	zipBuf := &bytes.Buffer{}
	zipWriter := zip.NewWriter(zipBuf)

	fp, _ := zipWriter.Create("README.txt")

    fp.Write([]byte{1, 2, 3}) // nolint: errcheck

	fp, _ = zipWriter.Create("db.bin")

    fp.Write([]byte{4, 5, 6}) // nolint: errcheck

	zipWriter.Close()

	httpmock.RegisterResponder("GET",
		"https://www.ip2location.com/download/",
		httpmock.NewBytesResponder(http.StatusOK, zipBuf.Bytes()))

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))

	content, err := ioutil.ReadFile(filepath.Join(suite.tmpDir, "database.bin"))

	suite.NoError(err)
	suite.Equal([]byte{4, 5, 6}, content)
}

func (suite *MockedIP2LocationTestSuite) TestOpenNothing() {
    suite.Error(suite.prov.Open(suite.tmpDir))
}

func (suite *MockedIP2LocationTestSuite) TestReopen() {
    suite.NoError(suite.prov.Open(suite.BaseDirectory()))
    suite.NoError(suite.prov.Open(suite.BaseDirectory()))

    ctx := context.Background()
    res, err := suite.prov.Lookup(ctx, net.ParseIP("80.80.80.80"))

    suite.NoError(err)
    suite.Equal("Amsterdam", res.City)
    suite.Equal("NL", res.CountryCode)

    suite.Error(suite.prov.Open(suite.tmpDir))

    res, err = suite.prov.Lookup(ctx, net.ParseIP("80.80.80.80"))

    suite.NoError(err)
    suite.Equal("Amsterdam", res.City)
    suite.Equal("NL", res.CountryCode)
}

type IntegrationIP2LocationTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationIP2LocationTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	prov, err := providers.NewIP2Location(suite.http,
		time.Minute,
		"",
		os.Getenv(ip2locationEnvApiKey),
		"")

	suite.NoError(err)

	suite.prov = prov
}

func (suite *IntegrationIP2LocationTestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.SetupTest()
	suite.TmpDirTestSuite.SetupTest()
}

func (suite *IntegrationIP2LocationTestSuite) TestFull() {
	ctx := context.Background()

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))
	suite.NoError(suite.prov.Open(suite.tmpDir))

	res, err := suite.prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

	suite.NoError(err)
	suite.Equal("Amsterdam", res.City)
	suite.Equal("NL", res.CountryCode)
}

func TestIP2Location(t *testing.T) {
	suite.Run(t, &MockedIP2LocationTestSuite{})
}

func TestIntegrationIP2Location(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
	}

	if os.Getenv(ip2locationEnvApiKey) == "" {
		t.Skip("Skipped because " + ip2locationEnvApiKey + " environment variable is empty")
	}

	suite.Run(t, &IntegrationIP2LocationTestSuite{})
}
