package providers

import "errors"

var (
	// ErrDatabaseIsNotReadyYet returns if you are trying to access
	// an offline provider but it haven't opened a database yet. For
	// example, it can be in process of downloading it.
	ErrDatabaseIsNotReadyYet = errors.New("database is not initialized yet")

	// ErrAuthTokenIsRequired is returned if you are trying to initialize
	// a provider which requires some token to work.
	ErrAuthTokenIsRequired = errors.New("auth token is required")

	// ErrNoFile is returned if provider has downloaded an archive with
	// database but this archive is empty.
	ErrNoFile = errors.New("cannot find a database file in downloaded archive")
)
