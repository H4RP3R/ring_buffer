package buffer

import (
	"fmt"
	"sync"
)

// Requesting is an enumeration of operations that can be requested from
// the ring buffer serving goroutine.
type Requesting string

const (
	// Pop is a request to remove and return an element from the beginning
	// of the buffer.
	Pop Requesting = "pop"
	// Get is a request to return an element from the beginning of the buffer
	// without removing it.
	Get Requesting = "get"
	// Clear is a request to fill the entire buffer storage with empty
	// variable values.
	Clear Requesting = "clear"
)

var (
	ErrInvalidBuffCap   = fmt.Errorf("buffer capacity is less than 1")
	ErrSendOnClosedChan = fmt.Errorf("send on closed channel")
)

type RingBuffer[T any] interface {
	// Push adds an element to the buffer. If the buffer is full, overwrites
	// the oldest element. If the element could not be placed, an error
	// is returned.
	Push(item T) error

	// Pop removes and returns an element from the beginning of the buffer.
	// If the buffer is empty, returns an empty value and false.
	Pop() (item T, ok bool)

	// IsEmpty checks if the buffer is empty.
	IsEmpty() bool

	// Full checks if the buffer is full.
	Full() bool

	// Size returns the current size of the buffer (number of elements).
	Size() int

	// Get returns an element from from the beginning of the buffer,
	// but does not remove it.
	Get() (item T, ok bool)

	// Clear clears the buffer, removing all elements.
	Clear()
}

// ringBuffer is a thread-safe ring buffer implementation.
type ringBuffer[T any] struct {
	data []T
	cap  int

	sizeMu sync.RWMutex
	size   int

	writerIdx     int
	readerIdx     int
	lastWriterIdx int
	wrapped       bool

	dataChan     chan T
	dataModified chan bool
	request      chan Requesting
}

// serve runs in a separate goroutine and provides thread-safe access to the
// ringBuffer data storage for other public and private buffer methods.
func (rb *ringBuffer[T]) serve() {
	for {
		select {
		case item, ok := <-rb.dataChan:
			if ok {
				rb.data[rb.writerIdx] = item
				rb.lastWriterIdx = rb.writerIdx
				if round := rb.shiftIdx(&rb.writerIdx); round {
					rb.wrapped = true
				}
				rb.sizeMu.Lock()
				if rb.size < rb.cap {
					rb.size++
				}
				rb.sizeMu.Unlock()
				rb.dataModified <- true
			}
		case requesting := <-rb.request:
			if requesting == Pop {
				item := rb.data[rb.readerIdx]
				rb.writeZeroVal(rb.readerIdx)
				rb.checkRightBound()
				if round := rb.shiftIdx(&rb.readerIdx); round {
					rb.wrapped = false
				}
				rb.dataModified <- true
				rb.dataChan <- item
			} else if requesting == Get {
				rb.dataChan <- rb.data[rb.readerIdx]
			} else if requesting == Clear {
				for i := 0; i < rb.cap; i++ {
					rb.writeZeroVal(i)
				}
				rb.dataModified <- true
			}
		}
	}
}

func (rb *ringBuffer[T]) Push(item T) (err error) {
	defer func() {
		// Catch the panic and return an error instead.
		if r := recover(); r != nil {
			err = fmt.Errorf("%w data channel is closed", ErrSendOnClosedChan)
		}
	}()
	rb.dataChan <- item
	<-rb.dataModified
	return err
}

func (rb *ringBuffer[T]) Pop() (T, bool) {
	if rb.Size() == 0 {
		var zero T
		return zero, false
	}

	rb.request <- Pop
	modified := <-rb.dataModified
	if !modified {
		var zero T
		return zero, false
	}
	item, ok := <-rb.dataChan
	return item, ok
}

func (rb *ringBuffer[T]) IsEmpty() bool {
	return rb.Size() == 0
}

func (rb *ringBuffer[T]) Size() int {
	rb.sizeMu.RLock()
	defer rb.sizeMu.RUnlock()
	return rb.size
}

func (rb *ringBuffer[T]) Get() (T, bool) {
	if rb.Size() == 0 {
		var zero T
		return zero, false
	}
	rb.request <- Get
	item, ok := <-rb.dataChan
	return item, ok
}

func (rb *ringBuffer[T]) Clear() {
	if rb.IsEmpty() {
		return
	}
	rb.request <- Clear
	<-rb.dataModified
}

// New returns a new thread-safe ring buffer with the given capacity and
// starts the essential goroutine that handles operations with the buffer data.
// If the specified capacity is less than 1, returns an error.
func New[T any](capacity int) (rb *ringBuffer[T], err error) {
	if capacity < 1 {
		return rb, ErrInvalidBuffCap
	}

	rb = &ringBuffer[T]{
		data:         make([]T, capacity),
		cap:          capacity,
		dataChan:     make(chan T),
		dataModified: make(chan bool),
		request:      make(chan Requesting),
	}

	go rb.serve()

	return rb, err
}

// writeZeroVal sets the element of the buffer data at the given index
// to the zero value of T.
func (rb *ringBuffer[T]) writeZeroVal(idx int) {
	var zero T
	rb.sizeMu.Lock()
	rb.data[idx] = zero
	if rb.size > 0 {
		rb.size--
	}
	rb.sizeMu.Unlock()
}

// checkRightBound verifies if the reader index has exceeded the valid data
// range and notifies via the dataModified channel.
func (rb *ringBuffer[T]) checkRightBound() {
	// Special case: prevent infinite wrapping when buffer capacity is 1.
	if rb.cap == 1 {
		rb.dataModified <- false
	}
	// Check if reader has exceeded the right bound after writer has wrapped around.
	if rb.wrapped && rb.readerIdx > rb.cap-1 {
		rb.dataModified <- false
	}
	// Check if reader has caught up with the writer (no wrap-around).
	if !rb.wrapped && rb.readerIdx >= rb.lastWriterIdx {
		rb.dataModified <- false
	}
}

// shiftIdx advances the index to the next position in the buffer, wrapping
// around to 0 if necessary. Returns true if the index was reset to 0,
// false otherwise.
func (rb *ringBuffer[T]) shiftIdx(idx *int) bool {
	if *idx < rb.cap-1 {
		*idx++
		return false
	} else {
		*idx = 0
		return true
	}
}
