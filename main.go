package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/9seconds/topographer/topolib"
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

	rootCtx, cancel := makeRootContext()
	defer cancel()

	if err := os.MkdirAll(conf.GetRootDirectory(), 0777); err != nil {
		return fmt.Errorf("cannot create root directory %s: %w", conf.GetRootDirectory(), err)
	}

	providers, err := makeProviders(conf)
	if err != nil {
		return fmt.Errorf("cannot initialise a list of providers: %w", err)
	}

	topo, err := topolib.NewTopographer(providers, newLogger(), conf.GetWorkerPoolSize())
	if err != nil {
		return fmt.Errorf("cannot initialize topographer: %w", err)
	}

	var httpHandler http.Handler = topo

	if conf.HasBasicAuth() {
		httpHandler = &basicAuthMiddleware{
			handler:  httpHandler,
			user:     conf.GetBasicAuthUser(),
			password: conf.GetBasicAuthPassword(),
		}
	}

	srv := &http.Server{
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
		Handler:      httpHandler,
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

	srv.Serve(listener) // nolint: errcheck

	<-closeChan

	return nil
}
