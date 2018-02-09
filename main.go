package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/9seconds/topographer/api"
	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/providers"
)

var (
	app = kingpin.New(
		"topographer",
		"Fast and lenient IP geolocation service.")

	debug = app.Flag("debug", "Run in debug mode.").
		Short('d').
		Envar("TOPOGRAPHER_DEBUG").
		Bool()
	host = app.Flag("host", "Host to bind to.").
		Short('b').
		Default("127.0.0.1").
		Envar("TOPOGRAPHER_HOST").
		String()
	port = app.Flag("port", "Port to bind to.").
		Short('p').
		Default("8000").
		Envar("TOPOGRAPHER_PORT").
		Int()
	configFile = app.Arg("config-path", "Path to the config.").
			Required().
			File()
)

func init() {
	app.Version("0.0.1")
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.WarnLevel)
}

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	hostPort := *host + ":" + strconv.Itoa(*port)

	conf, err := config.Parse(*configFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	pset := providers.NewProviderSet(conf)
	ticker := time.NewTicker(conf.UpdateEach.Duration)
	go func() {
		pset.Update(true)
		for range ticker.C {
			pset.Update(false)
		}
	}()

	router := api.MakeServer(pset)
	if err := http.ListenAndServe(hostPort, router); err != nil {
		log.Fatalf(err.Error())
	}
}
