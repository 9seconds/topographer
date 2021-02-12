package topolib

import (
	"encoding/json"
	"sync"
	"time"
)

// UsageStats collects different counters which could be useful to
// detect providers which are in used or stale.
//
// Currently we write a timestamp of the last usage, timestamp of last
// update (sucessful, only for offline providers); counters for a number
// of success and failed lookups.
type UsageStats struct {
    // A name of the provider.
	Name string

	mutex        sync.Mutex
	lastUpdated  time.Time
	lastUsed     time.Time
	successCount uint64
	failureCount uint64
}

func (u *UsageStats) notifyUsed(err error) {
	now := time.Now()

	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.lastUsed = now

	if err == nil {
		u.successCount += 1
	} else {
		u.failureCount += 1
	}
}

func (u *UsageStats) notifyUpdated() {
	now := time.Now()

	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.lastUpdated = now
}

// MarshalJSON to conform json.Marshaller interface.
func (u *UsageStats) MarshalJSON() ([]byte, error) {
	var lastUpdatedTime, lastUsedTime int64

	u.mutex.Lock()

	if !u.lastUpdated.IsZero() {
		lastUpdatedTime = u.lastUpdated.Unix()
	}

	if !u.lastUsed.IsZero() {
		lastUsedTime = u.lastUsed.Unix()
	}

	rawStruct := struct {
		Name         string `json:"name"`
		LastUpdated  int64  `json:"last_updated"`
		LastUsed     int64  `json:"last_used"`
		SuccessCount uint64 `json:"success_count"`
		FailureCount uint64 `json:"failure_count"`
	}{
		Name:         u.Name,
		LastUpdated:  lastUpdatedTime,
		LastUsed:     lastUsedTime,
		SuccessCount: u.successCount,
		FailureCount: u.failureCount,
	}

	u.mutex.Unlock()

	return json.Marshal(&rawStruct)
}
