package providers_test

// const envApiKey = "MAXMIND_API_KEY"

// type MaxmindLiteTestSuite struct {
// 	OfflineProviderTestSuite
// 	HTTPMockMixin
// }

// type IntegrationMaxmindLiteTestSuite struct {
// 	OfflineProviderTestSuite
// }

// func (suite *IntegrationMaxmindLiteTestSuite) TestFull() {
// 	prov := providers.NewMaxmindLite(suite.http, time.Minute, "", map[string]string{
// 		"license_key": os.Getenv(envApiKey),
// 	})
//     fs := afero.NewBasePathFs(afero.NewMemMapFs(), "/").(*afero.BasePathFs)

//     suite.NoError(prov.Download(context.Background(), afero.Afero{Fs: fs}))
// 	suite.NoError(prov.Open(fs))

// 	_, err := prov.Lookup(context.Background(), net.ParseIP("80.80.80.80"))

// 	suite.NoError(err)
// }

// func TestMaxmindLite(t *testing.T) {
// 	suite.Run(t, &MaxmindLiteTestSuite{})
// }

// func TestIntegrationMaxmindLite(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipped because of the short mode")
// 		return
// 	}

// 	if os.Getenv(envApiKey) == "" {
// 		t.Skip("Skipped because " + envApiKey + " environment variable is empty")
// 		return
// 	}

// 	suite.Run(t, &IntegrationMaxmindLiteTestSuite{})
// }
