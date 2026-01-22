package observer

import "errors"

var (
	// ErrBufferFull is returned when the event buffer is full and DropOnFull is true
	ErrBufferFull = errors.New("event buffer is full")

	// ErrObserverClosed is returned when trying to emit to a closed observer
	ErrObserverClosed = errors.New("observer is closed")
)
