package buffer

import (
	"fmt"
	"sync"
)

type RingBuffer[T any] interface {
	Push(item T)
	TryPush(item T) error
	Pop() (T, bool)
	IsEmpty() bool
	IsFull() bool
	Size() int
	Get() (T, bool)
	Clear()
	DeepClear()
}

var ErrInvalidBuffCap = fmt.Errorf("buffer capacity is less than 1")
var ErrBufferIsFull = fmt.Errorf("buffer is full")

// ringBuffer is a thread-safe ring buffer implementation.
type ringBuffer[T any] struct {
	mu   sync.RWMutex
	data []T
	size int
	cap  int

	writerIdx     int
	readerIdx     int
	lastWriterIdx int
	wrapped       bool
}

// Push adds an element to the buffer. If the buffer is full, overwrites the
// oldest element. If the element could not be placed, an error is returned.
func (rb *ringBuffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.data[rb.writerIdx] = item
	rb.lastWriterIdx = rb.writerIdx
	if rb.size < cap(rb.data) {
		rb.size++
	}
	if round := rb.shiftIdx(&rb.writerIdx); round {
		rb.wrapped = true
	}
}

// TryPush attempts to add an element to the ring buffer. If the buffer is
// full, it returns ErrBufferFull without adding the element. If there is free
// space, it adds the element and returns nil.
func (rb *ringBuffer[T]) TryPush(item T) (err error) {
	if rb.IsFull() {
		return ErrBufferIsFull
	}

	rb.Push(item)
	return nil
}

// Pop removes and returns an element from the beginning of the buffer.
// If the buffer is empty, returns an empty value and false.
func (rb *ringBuffer[T]) Pop() (T, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.size == 0 {
		var zero T
		return zero, false
	}

	item := rb.data[rb.readerIdx]
	rb.writeZeroVal(rb.readerIdx)
	if round := rb.shiftIdx(&rb.readerIdx); round {
		rb.wrapped = false
	}
	return item, true
}

// IsEmpty checks if the buffer is empty.
func (rb *ringBuffer[T]) IsEmpty() bool {
	return rb.Size() == 0
}

// IsFull checks if the buffer is full.
func (rb *ringBuffer[T]) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size == cap(rb.data)
}

// Size returns the current size of the buffer (number of elements).
func (rb *ringBuffer[T]) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Get returns an element from from the beginning of the buffer,
// but does not remove it.
func (rb *ringBuffer[T]) Get() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	if rb.size == 0 {
		var zero T
		return zero, false
	}
	return rb.data[rb.readerIdx], true
}

// Clear resets the buffer to its initial state, removing all elements.
// This operation does not modify the underlying data and is a lightweight way
// to reuse the buffer.
func (rb *ringBuffer[T]) Clear() {
	if rb.IsEmpty() {
		return
	}
	rb.mu.Lock()
	rb.writerIdx = 0
	rb.readerIdx = 0
	rb.lastWriterIdx = 0
	rb.wrapped = false
	rb.size = 0
	rb.mu.Unlock()
}

// DeepClear erases all data in the buffer by writing zero values to all buffer
// cells. This operation has a time complexity of O(n), where n is the buffer
// size. Use this method when security or data sensitivity is a concern.
func (rb *ringBuffer[T]) DeepClear() {
	if rb.IsEmpty() {
		return
	}
	rb.mu.Lock()
	for i := 0; i < cap(rb.data); i++ {
		rb.writeZeroVal(i)
	}
	rb.mu.Unlock()
}

// New returns a new thread-safe ring buffer with the given capacity.
// If the specified capacity is less than 1, returns an error.
func New[T any](capacity int) (rb *ringBuffer[T], err error) {
	if capacity < 1 {
		return rb, ErrInvalidBuffCap
	}

	rb = &ringBuffer[T]{
		data: make([]T, capacity),
		cap:  capacity,
	}

	return rb, err
}

// writeZeroVal sets the element of the buffer data at the given index
// to the zero value of T.
func (rb *ringBuffer[T]) writeZeroVal(idx int) {
	var zero T
	rb.data[idx] = zero
	if rb.size > 0 {
		rb.size--
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
