package providers_test

import (
	"bytes"
	"compress/gzip"
	"context"
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

type MockedDBIPTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedDBIPTestSuite) BaseDirectory() string {
	return filepath.Join(suite.GetTestdataPath(), "maxmind")
}

func (suite *MockedDBIPTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewDBIPLite(suite.http, time.Minute, suite.BaseDirectory())
}

func (suite *MockedDBIPTestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *MockedDBIPTestSuite) TestName() {
	suite.Equal(providers.NameDBIPLite, suite.prov.Name())
}

func (suite *MockedDBIPTestSuite) TestUpdateEvery() {
	suite.Equal(time.Minute, suite.prov.UpdateEvery())
}

func (suite *MockedDBIPTestSuite) TestBaseDirectory() {
	suite.Equal(suite.BaseDirectory(), suite.prov.BaseDirectory())
}

func (suite *MockedDBIPTestSuite) TestDownloadCancelledContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageBadStatus() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageBadHTML() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, "<"))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageNoLink() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, `
<html><body>
    <div class="card">
        <div>
            <a href="url">Download me</a>
        </div>
    </div>
</body></html>
        `))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageNoChecksum() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, `
<html><body>
    <div class="card">
        <dl>
            <dd class="small">aaa</dd>
            <dd class="small"></dd>
        </dl>
        <div>
            <a href="https://download.db-ip.com/free/file.mmdb.gz" class="free_download_link">Download me</a>
        </div>
    </div>
</body></html>
        `))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestCannotDownloadFile() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, `
<html><body>
    <div class="card">
        <dl>
            <dd class="small">aaa</dd>
            <dd class="small">AAf4c61ddcc5e8a2dabede0f3b482cd9aea9434d</dd>
        </dl>
        <div>
            <a href="https://download.db-ip.com/free/file.mmdb.gz" class="free_download_link">Download me</a>
        </div>
    </div>
</body></html>
        `))
	httpmock.RegisterResponder("GET",
		"https://download.db-ip.com/free/file.mmdb.gz",
		httpmock.NewStringResponder(http.StatusNotFound, ""))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadCannotSaveFile() {
	if err := os.Chmod(suite.tmpDir, 0400); err != nil {
		panic(err)
	}

	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, `
<html><body>
    <div class="card">
        <dl>
            <dd class="small">aaa</dd>
            <dd class="small">AAf4c61ddcc5e8a2dabede0f3b482cd9aea9434d</dd>
        </dl>
        <div>
            <a href="https://download.db-ip.com/free/file.mmdb.gz" class="free_download_link">Download me</a>
        </div>
    </div>
</body></html>
        `))

	fileBuffer := &bytes.Buffer{}
	wr := gzip.NewWriter(fileBuffer)

	wr.Write([]byte{1, 2, 3}) // nolint: errcheck
	wr.Close()

	httpmock.RegisterResponder("GET",
		"https://download.db-ip.com/free/file.mmdb.gz",
		httpmock.NewBytesResponder(http.StatusOK, fileBuffer.Bytes()))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadIncorrectChecksum() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, `
<html><body>
    <div class="card">
        <dl>
            <dd class="small">aaa</dd>
            <dd class="small">7037807198c22a7d2b0807371d763779a84fdfca</dd>
        </dl>
        <div>
            <a href="https://download.db-ip.com/free/file.mmdb.gz" class="free_download_link">Download me</a>
        </div>
    </div>
</body></html>
        `))

	fileBuffer := &bytes.Buffer{}
	wr := gzip.NewWriter(fileBuffer)

	wr.Write([]byte{1, 2, 3}) // nolint: errcheck
	wr.Close()

	httpmock.RegisterResponder("GET",
		"https://download.db-ip.com/free/file.mmdb.gz",
		httpmock.NewBytesResponder(http.StatusOK, fileBuffer.Bytes()))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MockedDBIPTestSuite) TestDownloadOk() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, `
<html><body>
    <div class="card">
        <dl>
            <dd class="small">aaa</dd>
            <dd class="small">7037807198c22a7d2b0807371d763779a84fdfcF</dd>
        </dl>
        <div>
            <a href="https://download.db-ip.com/free/file.mmdb.gz" class="free_download_link">Download me</a>
        </div>
    </div>
</body></html>
        `))

	fileBuffer := &bytes.Buffer{}
	wr := gzip.NewWriter(fileBuffer)

	wr.Write([]byte{1, 2, 3}) // nolint: errcheck
	wr.Close()

	httpmock.RegisterResponder("GET",
		"https://download.db-ip.com/free/file.mmdb.gz",
		httpmock.NewBytesResponder(http.StatusOK, fileBuffer.Bytes()))

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))
}

type IntegrationDBIPTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationDBIPTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()
}

func (suite *IntegrationDBIPTestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *IntegrationDBIPTestSuite) TestFull() {
	prov := providers.NewDBIPLite(suite.http, time.Minute, "")

	suite.NoError(prov.Download(context.Background(), suite.tmpDir))
	suite.NoError(prov.Open(suite.tmpDir))

	_, err := prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

	suite.NoError(err)
}

func TestDBIP(t *testing.T) {
	suite.Run(t, &MockedDBIPTestSuite{})
}

func TestIntegrationDBIP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
		return
	}

	suite.Run(t, &IntegrationDBIPTestSuite{})
}
