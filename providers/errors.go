package providers

import "errors"

var (
	ErrDatabaseIsNotReadyYet = errors.New("database is not initialized yet")
)
