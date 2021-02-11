package topolib_test

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/stretchr/testify/suite"
)

type usageStatsJSON struct {
	Name         string `json:"name"`
	LastUpdated  int64  `json:"last_updated"`
	LastUsed     int64  `json:"last_used"`
	SuccessCount uint64 `json:"success_count"`
	FailureCount uint64 `json:"failure_count"`
}

type UsageStatsTestSuite struct {
	suite.Suite

	u *topolib.UsageStats
}

func (suite *UsageStatsTestSuite) SetupTest() {
	suite.u = &topolib.UsageStats{
		Name: "test",
	}
}

func (suite *UsageStatsTestSuite) VerifyTime(expected time.Time, actual int64) {
    if expected.IsZero() {
        suite.EqualValues(0, actual)
    } else {
		suite.WithinDuration(expected, time.Unix(actual, 0), time.Second)
    }
}

func (suite *UsageStatsTestSuite) Verify(lastUsed, lastUpdated time.Time,
	success, failure int) {
	v, err := json.Marshal(suite.u)

	suite.NoError(err)

	raw := usageStatsJSON{}

	suite.NoError(json.Unmarshal(v, &raw))
	suite.Equal("test", raw.Name)
	suite.EqualValues(success, raw.SuccessCount)
	suite.EqualValues(failure, raw.FailureCount)
    suite.VerifyTime(lastUsed, raw.LastUsed)
    suite.VerifyTime(lastUpdated, raw.LastUpdated)
}

func (suite *UsageStatsTestSuite) TestEmpty() {
	suite.Verify(time.Time{}, time.Time{}, 0, 0)
}

func (suite *UsageStatsTestSuite) TestUsed() {
	suite.u.Used(nil)
	suite.Verify(time.Now(), time.Time{}, 1, 0)

	suite.u.Used(io.EOF)
	suite.Verify(time.Now(), time.Time{}, 1, 1)

	suite.u.Used(io.EOF)
	suite.Verify(time.Now(), time.Time{}, 1, 2)

	suite.u.Used(nil)
	suite.Verify(time.Now(), time.Time{}, 2, 2)
}

func (suite *UsageStatsTestSuite) TestUpdated() {
    suite.u.Updated()
	suite.Verify(time.Time{}, time.Now(), 0, 0)
}

func TestUsageStats(t *testing.T) {
	suite.Run(t, &UsageStatsTestSuite{})
}
