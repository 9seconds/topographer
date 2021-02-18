package main

import (
	"net"
	"os"

	"github.com/9seconds/topographer/topolib"
	"github.com/rs/zerolog"
)

type logger struct {
	lookupLog zerolog.Logger
	updateLog zerolog.Logger
}

func (l *logger) LookupError(ip net.IP, name string, err error) {
	l.lookupLog.Error().Str("provider", name).Stringer("ip", ip).Err(err).Msg("")
}

func (l *logger) UpdateInfo(name string) {
	l.updateLog.Info().Str("provider", name).Msg("Database was updated")
}

func (l *logger) UpdateError(name string, err error) {
	l.updateLog.Error().Str("provider", name).Err(err).Msg("")
}

func newLogger() topolib.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	return &logger{
		lookupLog: zerolog.New(os.Stderr).With().Timestamp().Stack().Str("event_name", "lookup").Logger(),
		updateLog: zerolog.New(os.Stderr).With().Timestamp().Stack().Str("event_name", "update").Logger(),
	}
}
