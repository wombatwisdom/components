package spec

import "errors"

var ErrAlreadyConnected = errors.New("already connected")
var ErrNotConnected = errors.New("not connected")
var ErrNoData = errors.New("no data available")
