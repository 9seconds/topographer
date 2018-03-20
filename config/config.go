package config

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
)

// ValidDatabases maps supported database number to 'some' value. It does not
// matter which value, map here is only for the faster lookup.
var ValidDatabases = map[string]bool{
	"dbip":        true,
	"ip2location": true,
	"maxmind":     true,
	"software77":  true,
	"sypex":       true,
}

type duration struct {
	time.Duration
}

func (dur *duration) UnmarshalText(text []byte) (err error) {
	dur.Duration, err = time.ParseDuration(string(text))
	return
}

// Precision is a type for geolocation precision, per country or per city
type Precision uint8

const (
	// PrecisionCountry defines geolocation precision up to country.
	PrecisionCountry = Precision(iota)

	// PrecisionCity defines geolocation precision up to city.
	PrecisionCity
)

// DBConfig is a mapping of configuration file section to the db config.
type DBConfig struct {
	Enabled bool
	Weight  float64
}

// Config is a configuration read from the file.
type Config struct {
	UpdateEach   duration `toml:"update_each"`
	Directory    string
	Databases    map[string]DBConfig
	PrecisionStr string `toml:"precision"`
	Precision    Precision
}

// Parse parses given configuration file and returns instance of the config.
func Parse(file io.Reader) (*Config, error) {
	conf := &Config{}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read config file")
	}

	if _, err = toml.Decode(string(buf), conf); err != nil {
		return nil, errors.Annotate(err, "Cannot parse config file")
	}

	if err = validate(conf); err != nil {
		return nil, errors.Annotate(err, "Invalid value")
	}

	return conf, nil
}

func validate(conf *Config) error { // nolint: gocyclo
	for k, v := range conf.Databases {
		if _, ok := ValidDatabases[k]; !ok {
			return errors.Errorf("Unknown database %s", k)
		}
		if v.Weight < 0.0 {
			return errors.Errorf("Incorrect weight %f for database %s",
				v.Weight, k)
		}
	}

	switch strings.ToLower(conf.PrecisionStr) {
	case "", "country":
		conf.Precision = PrecisionCountry
	case "city":
		conf.Precision = PrecisionCity
	default:
		return errors.Errorf("Unsupported value for precision.")
	}

	if conf.Directory == "" {
		path, err := os.Getwd()
		if err != nil {
			return errors.Annotate(err, "Cannot read current directory")
		}
		conf.Directory = path
	}
	if conf.UpdateEach.Duration == time.Duration(0) {
		conf.UpdateEach.Duration = 6 * time.Hour
	}

	if stat, err := os.Stat(conf.Directory); err != nil {
		return errors.Annotatef(err, "Incorrect directory %s", conf.Directory)
	} else if !stat.IsDir() {
		return errors.Annotatef(err, "Incorrect directory %s", conf.Directory)
	}

	return nil
}
