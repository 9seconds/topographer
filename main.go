package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/9seconds/topographer/topographer"
	"github.com/leaanthony/clir"
)

var version = "dev"

var (
	configPath = ""

	cli = clir.NewCli("topographer", "A lenient IP geolocation service", version)

	errNoConfigPath = errors.New("need to set a config path")
)

const (
	DefaultReadTimeout  = 10 * time.Second
	DefaultWriteTimeout = 10 * time.Second
)

func main() {
	cli.StringFlag("config", "A path to config file", &configPath)
	cli.Action(mainFunc)

	if err := cli.Run(); err != nil {
		panic(err)
	}
}

func mainFunc() error {
	if configPath == "" {
		return errNoConfigPath
	}

	conf, err := parseConfig(configPath)
	if err != nil {
		return fmt.Errorf("cannot read config: %w", err)
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)

	go func() {
		for range sigChan {
			cancel()
		}
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		Handler: topographer.Handler(topographer.Opts{
			Context: rootCtx,
			Providers: []topographer.Provider{
				topographer.TestProvider{},
				topographer.TestProvider{},
				topographer.TestProvider{},
			},
		}),
	}
	closeChan := make(chan struct{})

	go func() {
		<-rootCtx.Done()
		srv.Shutdown(context.Background()) // nolint: errcheck
		close(closeChan)
	}()

	listener, err := net.Listen("tcp", conf.Listen)
	if err != nil {
		return fmt.Errorf("cannot start listener: %w", err)
	}

	defer listener.Close()

	if err := srv.Serve(listener); err != nil {
		return fmt.Errorf("stopped to manage requests: %w", err)
	}

	<-closeChan

	return nil
}
