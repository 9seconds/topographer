package providers_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

type MockedSoftware77TestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedSoftware77TestSuite) BaseDirectory() string {
	return filepath.Join(suite.GetTestdataPath(), "software77")
}

func (suite *MockedSoftware77TestSuite) GetCSVResponses(value int) (string, []byte) {
	buf := &bytes.Buffer{}
	hasher := md5.New()
	gzipWriter := gzip.NewWriter(buf)
	writer := io.MultiWriter(hasher, gzipWriter)
	csvWriter := csv.NewWriter(writer)

	csvWriter.Write([]string{strconv.Itoa(value), strconv.Itoa(value + 1)})         // nolint: errcheck
	csvWriter.Write([]string{strconv.Itoa(10 * value), strconv.Itoa(10*value + 1)}) // nolint: errcheck
	csvWriter.Flush()
	gzipWriter.Close()

	return hex.EncodeToString(hasher.Sum(nil)), buf.Bytes()
}

func (suite *MockedSoftware77TestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewSoftware77(suite.http,
		time.Minute,
		suite.BaseDirectory())
}

func (suite *MockedSoftware77TestSuite) TearDownTest() {
	suite.HTTPMockMixin.TearDownTest()
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *MockedSoftware77TestSuite) TestName() {
	suite.Equal(providers.NameSoftware77, suite.prov.Name())
}

func (suite *MockedSoftware77TestSuite) TestUpdateEvery() {
	suite.Equal(time.Minute, suite.prov.UpdateEvery())
}

func (suite *MockedSoftware77TestSuite) TestBaseDirectory() {
	suite.Equal(suite.BaseDirectory(), suite.prov.BaseDirectory())
}

func (suite *MockedSoftware77TestSuite) TestDownloadCancelledContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadIPv4MD5UrlNotFound() {
	ctx := context.Background()
	_, v4Data := suite.GetCSVResponses(1)
	v6Checksum, v6Data := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusNotFound, ""))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusOK, v6Checksum))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadIPv6MD5UrlNotFound() {
	ctx := context.Background()
	v4Checksum, v4Data := suite.GetCSVResponses(1)
	_, v6Data := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, v4Checksum))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewBytesResponder(http.StatusNotFound, nil))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadIPv4FileUrlNotFound() {
	ctx := context.Background()
	v4Checksum, _ := suite.GetCSVResponses(1)
	v6Checksum, v6Data := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusNotFound, nil))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, v4Checksum))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusOK, v6Checksum))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadIPv6FileUrlNotFound() {
	ctx := context.Background()
	v4Checksum, v4Data := suite.GetCSVResponses(1)
	v6Checksum, _ := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, v4Checksum))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, nil))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusNotFound, v6Checksum))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadBrokenChecksumNotFound() {
	ctx := context.Background()
	_, v4Data := suite.GetCSVResponses(1)
	v6Checksum, v6Data := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, "xxx"))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusOK, v6Checksum))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadFSReadOnly() {
	ctx := context.Background()
	v4Checksum, v4Data := suite.GetCSVResponses(1)
	v6Checksum, v6Data := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, v4Checksum))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusOK, v6Checksum))

	suite.NoError(os.Chmod(suite.tmpDir, 0555))
	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadChecksumMismatch() {
	ctx := context.Background()
	_, v4Data := suite.GetCSVResponses(1)
	v6Checksum, v6Data := suite.GetCSVResponses(2)
	checksum, _ := suite.GetCSVResponses(3)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, checksum))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusOK, v6Checksum))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestDownloadOk() {
	ctx := context.Background()
	v4Checksum, v4Data := suite.GetCSVResponses(1)
	v6Checksum, v6Data := suite.GetCSVResponses(2)

	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=1",
		httpmock.NewBytesResponder(http.StatusOK, v4Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=3",
		httpmock.NewStringResponder(http.StatusOK, v4Checksum))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=9",
		httpmock.NewBytesResponder(http.StatusOK, v6Data))
	httpmock.RegisterResponder("GET", "https://software77.net/geo-ip/?DL=10",
		httpmock.NewStringResponder(http.StatusOK, v6Checksum))

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedSoftware77TestSuite) TestLookupNotReady() {
	_, err := suite.prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

	suite.Error(err)
}

func (suite *MockedSoftware77TestSuite) TestLookupUnknownYet() {
	suite.NoError(suite.prov.Open(suite.BaseDirectory()))

	_, err := suite.prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

	suite.Error(err)
}

func (suite *MockedSoftware77TestSuite) TestLookupOk() {
	suite.NoError(suite.prov.Open(suite.BaseDirectory()))

	res, err := suite.prov.Lookup(context.Background(), net.ParseIP("1.0.128.2"))

	suite.NoError(err)
	suite.Equal("TH", res.CountryCode.String())
}

func (suite *MockedSoftware77TestSuite) TestLookupFaulyReopen() {
	suite.NoError(suite.prov.Open(suite.BaseDirectory()))
	suite.Error(suite.prov.Open(suite.tmpDir))

	res, err := suite.prov.Lookup(context.Background(), net.ParseIP("1.0.128.2"))

	suite.NoError(err)
	suite.Equal("TH", res.CountryCode.String())
}

type IntegrationSoftware77TestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationSoftware77TestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewSoftware77(suite.http,
		time.Minute,
		"")
}

func (suite *IntegrationSoftware77TestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *IntegrationSoftware77TestSuite) TestFull() {
	ctx := context.Background()

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))
	suite.NoError(suite.prov.Open(suite.tmpDir))
	suite.NoError(suite.prov.Open(suite.tmpDir))

	res, err := suite.prov.Lookup(ctx, net.ParseIP("80.80.80.80"))

	suite.NoError(err)
	suite.Equal("NL", res.CountryCode.String())
}

func TestSoftware77(t *testing.T) {
	suite.Run(t, &MockedSoftware77TestSuite{})
}

func TestIntegrationSoftware77(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
	}

	suite.Run(t, &IntegrationSoftware77TestSuite{})
}
