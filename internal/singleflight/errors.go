package singleflight

import "errors"

// ErrInProgress is returned by TryDo when another call with the same key
// is already in progress.
var ErrInProgress = errors.New("singleflight: call already in progress")