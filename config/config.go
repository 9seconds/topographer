package config

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
)

var VALID_DATABASES = map[string]bool{
	"dbip":        true,
	"ip2location": true,
	"maxmind":     true,
	"sypex":       true,
}

type duration struct {
	time.Duration
}

func (dur *duration) UnmarshalText(text []byte) (err error) {
	dur.Duration, err = time.ParseDuration(string(text))
	return
}

type Precision uint8

const (
	PRECISION_COUNTRY = Precision(iota)
	PRECISION_CITY
)

type DBConfig struct {
	Enabled bool
	Weight  float64
}

type Config struct {
	Host         string
	Port         int
	UpdateEach   duration `toml:"update_each"`
	Directory    string
	Databases    map[string]DBConfig
	PrecisionStr string `toml:"precision"`
	Precision    Precision
}

func Parse(file *os.File) (*Config, error) {
	conf := &Config{}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read config file")
	}

	if _, err := toml.Decode(string(buf), conf); err != nil {
		return nil, errors.Annotate(err, "Cannot parse config file")
	}

	if err = validate(conf); err != nil {
		return nil, errors.Annotate(err, "Invalid value")
	}

	return conf, nil
}

func validate(conf *Config) error {
	for k, v := range conf.Databases {
		if _, ok := VALID_DATABASES[k]; !ok {
			return errors.Errorf("Unknown database %s", k)
		}
		if v.Weight < 0.0 {
			return errors.Errorf("Incorrect weight %f for database %s",
				v.Weight, k)
		}
	}

	switch strings.ToLower(conf.PrecisionStr) {
	case "", "country":
		conf.Precision = PRECISION_COUNTRY
	case "city":
		conf.Precision = PRECISION_CITY
	default:
		return errors.Errorf("Unsupported value for precision.")
	}

	if conf.Host == "" {
		conf.Host = "127.0.0.1"
	}
	if conf.Port == 0 {
		conf.Port = 8000
	}
	if conf.Directory == "" {
		path, _ := os.Getwd()
		conf.Directory = path
	}
	if conf.UpdateEach.Duration == time.Duration(0) {
		conf.UpdateEach.Duration = 6 * time.Hour
	}

	if stat, err := os.Stat(conf.Directory); err != nil {
		return errors.Annotatef(err, "Incorrect directory %s", conf.Directory)
	} else {
		if !stat.IsDir() {
			return errors.Annotatef(err, "Incorrect directory %s", conf.Directory)
		}
	}

	return nil
}
