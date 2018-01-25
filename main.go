package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/providers/maxmind"
)

var (
	app = kingpin.New(
		"topographer",
		"Fast and lenient IP geolocation service")

	debug = app.Flag("debug", "Run in debug mode.").
		Short('d').
		Envar("TOPOGRAPHER_DEBUG").
		Bool()
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

	conf, err := config.Parse(*configFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Println(conf)
	mm := maxmind.NewProvider(conf)
	fmt.Println(mm.Update())
}
