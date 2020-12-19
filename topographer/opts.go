package topographer

import "context"

type Opts struct {
	Context       context.Context
	RootDirectory string
	Logger        Logger
	Providers     []Provider
}
