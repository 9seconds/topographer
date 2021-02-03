package providers

import "errors"

var (
	ErrDatabaseIsNotReadyYet = errors.New("database is not initialized yet")
	ErrAuthTokenIsRequired   = errors.New("auth token is required")
	ErrNoFile                = errors.New("cannot find a database file in downloaded archive")
)
