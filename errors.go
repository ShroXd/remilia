package remilia

import "errors"

var ErrInvalidInputBufferSize = errors.New("invalid input buffer size")
var ErrInvalidConcurrency = errors.New("invalid concurrency")
var ErrInvalidTimeout = errors.New("invalid timeout")
