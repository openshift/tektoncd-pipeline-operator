package manifestival

import "github.com/go-logr/logr"

type options struct {
	recursive bool
	logger    logr.Logger
	client    Client
}

type Option func(*options)

func Recursive(opts *options) {
	opts.recursive = true
}

func UseRecursive(v bool) Option {
	return func(opts *options) {
		opts.recursive = v
	}
}

func UseLogger(log logr.Logger) Option {
	return func(o *options) {
		o.logger = log
	}
}

func UseClient(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}
