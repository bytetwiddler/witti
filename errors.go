package witti

import "errors"

var (
	ErrEmptyQuery             = errors.New("empty query")
	ErrMultipleProjectedTimes = errors.New("multiple projected local times provided; use only one datetime argument")
	ErrInvalidRequest         = errors.New("invalid request")
)
