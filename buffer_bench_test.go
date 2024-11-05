package buffer

import (
	"fmt"
	"sync"
	"testing"
)

func BenchmarkRingBufferPush(b *testing.B) {
	bufCapacity := 2048
	buffer, err := New[int](bufCapacity)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Push(i)
	}
}

func BenchmarkRingBufferPushConcurrent(b *testing.B) {
	bufCapacity := 2048
	buffer, err := New[int](bufCapacity)
	if err != nil {
		b.Error(err)
	}

	gorAmount := 100
	var wg sync.WaitGroup
	wg.Add(gorAmount)

	b.ResetTimer()
	for i := 0; i < gorAmount; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < b.N; j++ {
				buffer.Push(j)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkRingBufferPop(b *testing.B) {
	bufCapacity := 2048
	buffer, err := New[int](bufCapacity)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < bufCapacity; i++ {
		buffer.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Pop()
	}
}

func BenchmarkRingBufferPopConcurrent(b *testing.B) {
	bufCapacity := 2048
	buffer, err := New[int](bufCapacity)
	if err != nil {
		b.Error(err)
	}

	gorAmount := 100
	var wg sync.WaitGroup
	wg.Add(gorAmount)

	b.ResetTimer()
	for i := 0; i < gorAmount; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < b.N; j++ {
				for {
					_, ok := buffer.Pop()
					if ok {
						break
					}
					// refill the buffer if it's empty
					buffer.Push(1)
				}
			}
		}()
	}

	wg.Wait()
}

func BenchmarkRingBufferClear(b *testing.B) {
	var testItem = struct{}{}
	testCases := []struct {
		bufCapacity int
		itemCount   int
	}{
		{bufCapacity: 100, itemCount: 50},
		{bufCapacity: 100, itemCount: 100},
		{bufCapacity: 1000, itemCount: 500},
		{bufCapacity: 1000, itemCount: 1000},
		{bufCapacity: 10_000, itemCount: 5000},
		{bufCapacity: 10_000, itemCount: 10_000},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("cap: %d, items: %d", tc.bufCapacity, tc.itemCount)
		b.Run(name, func(b *testing.B) {
			buffer, err := New[struct{}](tc.bufCapacity)
			if err != nil {
				b.Error(err)
			}

			iterations := 100_000
			// Using a constant number of iterations instead of b.N to avoid
			// the benchmarking framework from adjusting b.N to higher values.
			// Since Clear() takes much less time than Push(), the test could
			// take a very long time.
			for i := 0; i < iterations; i++ {
				// Refill the buffer
				for j := 0; j < tc.itemCount; j++ {
					buffer.Push(testItem)
				}
				b.ResetTimer()
				buffer.Clear()
			}
		})
	}
}
