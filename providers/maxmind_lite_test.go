package providers_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

const envApiKey = "MAXMIND_API_KEY"

type MaxmindLiteTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
	HTTPMockMixin
}

func (suite *MaxmindLiteTestSuite) BaseDirectory() string {
	return filepath.Join(suite.GetTestdataPath(), "maxmind")
}

func (suite *MaxmindLiteTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()

	suite.prov = providers.NewMaxmindLite(suite.http, time.Minute, suite.tmpDir, "apikey")
}

func (suite *MaxmindLiteTestSuite) TearDownSuite() {
	suite.HTTPMockMixin.TearDownTest()
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *MaxmindLiteTestSuite) TestName() {
	suite.Equal(providers.NameMaxmindLite, suite.prov.Name())
}

func (suite *MaxmindLiteTestSuite) TestUpdateEvery() {
	suite.Equal(time.Minute, suite.prov.UpdateEvery())
}

func (suite *MaxmindLiteTestSuite) TestBaseDirectory() {
	suite.Equal(suite.tmpDir, suite.prov.BaseDirectory())
}

func (suite *MaxmindLiteTestSuite) TestDownloadCancelledContext() {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotDownloadChecksumBadStatus() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotDownloadChecksumBadResponseFormat() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK, "???"))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotDownloadArchive() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK,
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824 GeoLite2-City.tar.gz"))
	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz",
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotDowloadArchiveChecksumMismatch() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK,
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824 GeoLite2-City.tar.gz"))
	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz",
		httpmock.NewStringResponder(http.StatusOK, ""))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotExtractArchiveNotGzip() {
	ctx := context.Background()

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK,
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824 GeoLite2-City.tar.gz"))
	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz",
		httpmock.NewStringResponder(http.StatusOK, "hello"))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotExtractArchiveNotTar() {
	ctx := context.Background()

	buf := &bytes.Buffer{}
	w := gzip.NewWriter(buf)
	hasher := sha256.New()

	w.Write([]byte("hello")) // nolint: errcheck
	w.Close()
	hasher.Write(buf.Bytes()) // nolint: errcheck

	hashed := hex.EncodeToString(hasher.Sum(nil))

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK,
			hashed+" GeoLite2-City.tar.gz"))
	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz",
		httpmock.NewBytesResponder(http.StatusOK, buf.Bytes()))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestCannotExtractArchiveNoFileInTar() {
	ctx := context.Background()

	buf := &bytes.Buffer{}
	w := gzip.NewWriter(buf)
	tarFile := tar.NewWriter(w)
	hasher := sha256.New()

	tarFile.WriteHeader(&tar.Header{ // nolint: errcheck
		Typeflag: tar.TypeReg,
		Name:     "file.txt",
		Mode:     0644,
		ModTime:  time.Now(),
		Size:     5,
	})
	tarFile.Write([]byte("hello")) // nolint: errcheck
	tarFile.Close()
	w.Close()
	hasher.Write(buf.Bytes()) // nolint: errcheck

	hashed := hex.EncodeToString(hasher.Sum(nil))

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK,
			hashed+" GeoLite2-City.tar.gz"))
	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz",
		httpmock.NewBytesResponder(http.StatusOK, buf.Bytes()))

	suite.Error(suite.prov.Download(ctx, suite.tmpDir))
}

func (suite *MaxmindLiteTestSuite) TestOk() {
	ctx := context.Background()

	buf := &bytes.Buffer{}
	w := gzip.NewWriter(buf)
	tarFile := tar.NewWriter(w)
	hasher := sha256.New()

	tarFile.WriteHeader(&tar.Header{ // nolint: errcheck
		Typeflag: tar.TypeReg,
		Name:     "file.mmdb",
		Mode:     0644,
		ModTime:  time.Now(),
		Size:     5,
	})
	tarFile.Write([]byte("hello")) // nolint: errcheck
	tarFile.Close()
	w.Close()
	hasher.Write(buf.Bytes()) // nolint: errcheck

	hashed := hex.EncodeToString(hasher.Sum(nil))

	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz.sha256",
		httpmock.NewStringResponder(http.StatusOK,
			hashed+" GeoLite2-City.tar.gz"))
	httpmock.RegisterResponder("GET",
		"https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=apikey&suffix=tar.gz",
		httpmock.NewBytesResponder(http.StatusOK, buf.Bytes()))

	suite.NoError(suite.prov.Download(ctx, suite.tmpDir))

	data, err := ioutil.ReadFile(filepath.Join(suite.tmpDir, "database.mmdb"))

	suite.NoError(err)
	suite.Equal([]byte("hello"), data)
}

type IntegrationMaxmindLiteTestSuite struct {
	TmpDirTestSuite
	OfflineProviderTestSuite
}

func (suite *IntegrationMaxmindLiteTestSuite) SetupTest() {
	suite.TmpDirTestSuite.SetupTest()
	suite.OfflineProviderTestSuite.SetupTest()
}

func (suite *IntegrationMaxmindLiteTestSuite) TearDownTest() {
	suite.OfflineProviderTestSuite.TearDownTest()
	suite.TmpDirTestSuite.TearDownTest()
}

func (suite *IntegrationMaxmindLiteTestSuite) TestFull() {
	prov := providers.NewMaxmindLite(suite.http, time.Minute, "", os.Getenv(envApiKey))

	suite.NoError(prov.Download(context.Background(), suite.tmpDir))
	suite.NoError(prov.Open(suite.tmpDir))

	_, err := prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

	suite.NoError(err)
}

func TestMaxmindLite(t *testing.T) {
	suite.Run(t, &MaxmindLiteTestSuite{})
}

func TestIntegrationMaxmindLite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipped because of the short mode")
		return
	}

	if os.Getenv(envApiKey) == "" {
		t.Skip("Skipped because " + envApiKey + " environment variable is empty")
		return
	}

	suite.Run(t, &IntegrationMaxmindLiteTestSuite{})
}
