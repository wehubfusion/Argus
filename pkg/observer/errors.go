package observer

import "errors"

var (
	// ErrObserverClosed is returned when trying to emit to a closed observer
	ErrObserverClosed = errors.New("observer is closed")
)
