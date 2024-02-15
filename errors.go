package remilia

import "errors"

var errInvalidInputBufferSize = errors.New("invalid input buffer size")
var errInvalidConcurrency = errors.New("invalid concurrency")
var errInvalidTimeout = errors.New("invalid timeout")
