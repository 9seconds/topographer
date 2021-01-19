package providers_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type MockedDBIPTestSuite struct {
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MockedDBIPTestSuite) BaseDirectory() string {
	path, err := filepath.Abs(filepath.Join("testdata", "maxmind"))
	if err != nil {
		panic(err)
	}

	return path
}

func (suite *MockedDBIPTestSuite) SetupTest() {
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewDBIPLite(suite.http, time.Minute, suite.BaseDirectory())
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
	fs := afero.NewMemMapFs()
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageBadStatus() {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageBadHTML() {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://db-ip.com/db/download/ip-to-city-lite",
		httpmock.NewStringResponder(http.StatusOK, "<"))

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageNoLink() {
	fs := afero.NewMemMapFs()
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

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadSeedPageNoChecksum() {
	fs := afero.NewMemMapFs()
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

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestCannotDownloadFile() {
	fs := afero.NewMemMapFs()
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

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadCannotSaveFile() {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
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

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadIncorrectChecksum() {
	fs := afero.NewMemMapFs()
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

	suite.Error(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func (suite *MockedDBIPTestSuite) TestDownloadOk() {
	fs := afero.NewMemMapFs()
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

	suite.NoError(suite.prov.Download(ctx, afero.Afero{Fs: fs}))
}

func TestDBIP(t *testing.T) {
	suite.Run(t, &MockedDBIPTestSuite{})
}
