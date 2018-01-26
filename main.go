package main

import (
	"fmt"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/providers"
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

	mm := providers.NewMaxMind(conf)
	i2l := providers.NewIP2Location(conf)
	sx := providers.NewSypex(conf)
	dbip := providers.NewDBIP(conf)
	// dbip.Update()
	fmt.Println(dbip.Reopen(time.Now()))
	// sx.Update()
	sx.Reopen(time.Now())
	// i2l.Update()
	// mm.Update()
	mm.Reopen(time.Now())
	i2l.Reopen(time.Now())
	fmt.Println(mm.Resolve([]net.IP{net.ParseIP("81.2.69.142")}))
	fmt.Println(i2l.Resolve([]net.IP{net.ParseIP("81.2.69.142")}))
	fmt.Println(sx.Resolve([]net.IP{net.ParseIP("93.73.35.74")}))
	fmt.Println(dbip.Resolve([]net.IP{net.ParseIP("93.73.35.74")}))
}
