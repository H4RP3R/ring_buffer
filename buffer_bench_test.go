package buffer

import (
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
