package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

const DefaultUpdateEvery = 24 * time.Hour

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var v interface{}

	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("cannot unmarshal duration: %w", err)
	}

	vv, ok := v.(string)
	if !ok {
		return fmt.Errorf("incorrect duration: %v", v)
	}

	dur, err := time.ParseDuration(vv)
	if err != nil {
		return fmt.Errorf("cannot parse duration: %w", err)
	}

	d.Duration = dur

	return nil
}

type config struct {
	Listen        string           `json:"listen"`
	RootDirectory string           `json:"root_directory"`
	Providers     []configProvider `json:"providers"`
}

type configProvider struct {
	Name        string   `json:"name"`
	Directory   string   `json:"directory"`
	UpdateEvery duration `json:"update_every"`
}

func parseConfig(path string) (*config, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	conf := config{}

	if err := json.Unmarshal(content, &conf); err != nil {
		return nil, fmt.Errorf("cannot parse json: %w", err)
	}

	if _, _, err := net.SplitHostPort(conf.Listen); err != nil {
		return nil, fmt.Errorf("incorrect host:port for listen: %w", err)
	}

	directory, err := os.OpenFile(conf.RootDirectory, os.O_RDWR|os.O_CREATE, 0777)

	switch {
	case os.IsNotExist(err):
		return nil, fmt.Errorf("directory %s is not exist", conf.RootDirectory)
	case os.IsPermission(err):
		return nil, fmt.Errorf("cannot open a directory with correct permissions: %w", err)
	}

	directory.Close()

	return &conf, nil
}
