package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigOk(t *testing.T) {
	text := `update_each = "6h"
		directory = "/tmp"
		precision = "city"

		[databases]

			[databases.ip2location]
			enabled = true
			weight = 1.0

			[databases.dbip]
			enabled = true
			weight = 1.1

			[databases.maxmind]
			enabled = true
			weight = 1.3

			[databases.sypex]
			enabled = true
			weight = 0.9`

	conf, err := Parse(strings.NewReader(text))
	assert.Nil(t, err)
	assert.NotNil(t, conf)

	dur, _ := time.ParseDuration("6h")
	assert.Equal(t, conf.UpdateEach.Duration, dur)
	assert.Equal(t, conf.Directory, "/tmp")
	assert.Equal(t, conf.Precision, PrecisionCity)

	for _, name := range []string{"ip2location", "dbip", "maxmind", "sypex"} {
		assert.Contains(t, conf.Databases, name)
		assert.True(t, conf.Databases[name].Enabled)
	}

	assert.InDelta(t, conf.Databases["ip2location"].Weight, 1.0, 1e-6)
	assert.InDelta(t, conf.Databases["dbip"].Weight, 1.1, 1e-6)
	assert.InDelta(t, conf.Databases["maxmind"].Weight, 1.3, 1e-6)
	assert.InDelta(t, conf.Databases["sypex"].Weight, 0.9, 1e-6)
}

func TestConfigDefaults(t *testing.T) {
	text := "[databases]"

	conf, err := Parse(strings.NewReader(text))
	assert.Nil(t, err)
	assert.NotNil(t, conf)

	dur, _ := time.ParseDuration("6h")
	assert.Equal(t, conf.UpdateEach.Duration, dur)

	path, _ := os.Getwd()
	assert.Equal(t, conf.Directory, path)

	assert.Equal(t, conf.Precision, PrecisionCountry)

	assert.Len(t, conf.Databases, 0)
}

func TestUnknownDatabase(t *testing.T) {
	text := `
		[databases]

			[databases.qqq]
			enabled = true
			weight = 1.0`

	_, err := Parse(strings.NewReader(text))
	assert.NotNil(t, err)
}

func TestIncorrectWeight(t *testing.T) {
	text := `
		[databases]

			[databases.maxmind]
			enabled = true
			weight = -1.0`

	_, err := Parse(strings.NewReader(text))
	assert.NotNil(t, err)
}

func TestIncorrectPrecision(t *testing.T) {
	text := `
		precision = "lalalal"
		[databases]

			[databases.maxmind]
			enabled = true
			weight = 1.0`

	_, err := Parse(strings.NewReader(text))
	assert.NotNil(t, err)
}

func TestIncorrectDirectory(t *testing.T) {
	text := `
		directory = "/assdlfkjhsdfkjshfkladflskafsalkfhaslg;f234r4fsd"
		[databases]

			[databases.maxmind]
			enabled = true
			weight = 1.0`

	_, err := Parse(strings.NewReader(text))
	assert.NotNil(t, err)
}
