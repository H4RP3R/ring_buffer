package buffer

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestRingBufferImplementsInterface(t *testing.T) {
	buffer, _ := New[string](1)
	checkInterfaceImplementation := func(rb interface{}) bool {
		_, ok := rb.(RingBuffer[string])
		return ok
	}
	if !checkInterfaceImplementation(buffer) {
		t.Errorf("ringBuffer does not implement RingBuffer interface")
	}
}

func TestRingBufferPushInt(t *testing.T) {
	testCases := []struct {
		bufCapacity int
		testItems   []int
		bufDataWant []int
	}{
		{
			bufCapacity: 1,
			testItems:   []int{},
			bufDataWant: []int{0},
		},
		{
			bufCapacity: 3,
			testItems:   []int{42},
			bufDataWant: []int{42, 0, 0},
		},
		{
			bufCapacity: 3,
			testItems:   []int{1, 2, 3},
			bufDataWant: []int{1, 2, 3},
		},
		{
			bufCapacity: 3,
			testItems:   []int{1, 2, 3, 4, 5, 6, 7},
			bufDataWant: []int{7, 5, 6},
		},
		{
			bufCapacity: 2,
			testItems:   []int{},
			bufDataWant: []int{0, 0},
		},
		{
			bufCapacity: 2,
			testItems:   []int{1, 2, 3},
			bufDataWant: []int{3, 2},
		},
		{
			bufCapacity: 2,
			testItems:   []int{1, 2, 3, 4, 5, 6},
			bufDataWant: []int{5, 6},
		},
		{
			bufCapacity: 5,
			testItems:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			bufDataWant: []int{6, 7, 8, 9, 5},
		},
		{
			bufCapacity: 6,
			testItems:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			bufDataWant: []int{13, 14, 15, 10, 11, 12},
		},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("cap: %d, items: %d", tc.bufCapacity, len(tc.testItems))
		t.Run(name, func(t *testing.T) {
			buffer, err := New[int](tc.bufCapacity)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range tc.testItems {
				buffer.Push(item)
			}

			if !reflect.DeepEqual(tc.bufDataWant, buffer.data) {
				t.Errorf("buffer items: want: %v, got %v", tc.bufDataWant, buffer.data)
			}

			wantBufSize := min(len(tc.testItems), tc.bufCapacity)
			if buffer.Size() != wantBufSize {
				t.Errorf("wrong buffer size: want %d, got %d", wantBufSize, buffer.Size())
			}
		})
	}
}

func TestRingBufferTryPushInt(t *testing.T) {
	testCases := []struct {
		name        string
		bufCapacity int
		testItems   []int
	}{
		{name: "1 item", bufCapacity: 1, testItems: []int{1}},
		{name: "10 items", bufCapacity: 10, testItems: []int{2, 4, 13, 13, 30, 45, 12, 7, 35, 19}},
		{name: "random items", bufCapacity: 1024, testItems: randomNumbers(1024, 0, 1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lastItem := 42
			buffer, err := New[int](tc.bufCapacity)
			if err != nil {
				t.Fatal(err)
			}

			// Fill buffer to capacity
			for _, item := range tc.testItems {
				err := buffer.TryPush(item)
				if err != nil {
					t.Errorf("didn't expect an error: %v", err)
				}
			}

			// TryPush to full buffer
			err = buffer.TryPush(lastItem)
			if !errors.Is(err, ErrBufferIsFull) {
				t.Errorf("expected err: %v, got err: %v", ErrBufferIsFull, err)
			}

			// Match buffer content
			expectedItems := tc.testItems
			if len(expectedItems) > tc.bufCapacity {
				expectedItems = expectedItems[len(expectedItems)-tc.bufCapacity:]
			}
			if !reflect.DeepEqual(buffer.data[:buffer.size], expectedItems) {
				t.Errorf("buffer data does not match test items")
			}

			// Pop and push new item
			buffer.Pop()
			err = buffer.TryPush(lastItem)
			if err != nil {
				t.Errorf("didn't expect an error: %v", err)
			}

			// Calculate the expected index for the last item
			expectedIndex := (buffer.writerIdx - 1 + cap(buffer.data)) % cap(buffer.data)
			// Verify last item is correctly placed
			if buffer.data[expectedIndex] != lastItem {
				t.Errorf("last item: want %d, got %d", lastItem, buffer.data[expectedIndex])
			}
		})
	}
}

func TestRingBufferPopString(t *testing.T) {
	testCases := []struct {
		bufCapacity int
		testItems   []string
		wantItems   []string
	}{
		{
			bufCapacity: 1,
			testItems:   []string{"apple"},
			wantItems:   []string{"apple"},
		},
		{
			bufCapacity: 5,
			testItems:   []string{"apple", "banana"},
			wantItems:   []string{"apple", "banana"},
		},
		{
			bufCapacity: 3,
			testItems:   []string{"apple", "banana", "orange", "pork", "tomato"},
			wantItems:   []string{"pork", "tomato", "orange"},
		},
		{
			bufCapacity: 2,
			testItems:   []string{"kiwi", "mango", "pear", "grape"},
			wantItems:   []string{"pear", "grape"},
		},
		{
			bufCapacity: 1,
			testItems:   []string{"lemon", "lime", "melon"},
			wantItems:   []string{"melon"},
		},
		{
			bufCapacity: 6,
			testItems:   []string{"strawberry", "blueberry", "raspberry", "blackberry", "cherry"},
			wantItems:   []string{"strawberry", "blueberry", "raspberry", "blackberry", "cherry"},
		},
		// Additional test case: empty buffer
		{
			bufCapacity: 3,
			testItems:   []string{},
			wantItems:   []string{},
		},
		// Additional test case: buffer with repeated elements
		{
			bufCapacity: 2,
			testItems:   []string{"apple", "apple", "apple"},
			wantItems:   []string{"apple", "apple"},
		},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("cap: %d, items: %d", tc.bufCapacity, len(tc.testItems))
		t.Run(name, func(t *testing.T) {
			buffer, err := New[string](tc.bufCapacity)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range tc.testItems {
				buffer.Push(item)
			}

			if buffer.Size() != len(tc.wantItems) {
				t.Errorf("wrong items amount")
			}

			for _, testItem := range tc.wantItems {
				if item, ok := buffer.Pop(); ok {
					if item != testItem {
						t.Errorf("item: want %v, got %v", testItem, item)
					}
				}
			}
		})
	}
}

func TestRingBufferPopFromEmptyBuffer(t *testing.T) {
	buffer, err := New[int](2)
	if err != nil {
		t.Fatal(err)
	}

	want := 0
	got, ok := buffer.Pop()
	if got != want {
		t.Errorf("want: %d, got: %d", want, got)
	}
	if ok != false {
		t.Errorf("expected ok: false, got ok: %t", ok)
	}
}

func TestRingBufferIsEmpty(t *testing.T) {
	testCases := []struct {
		bufCapacity int
		itemsAmount int
		want        bool
	}{
		{bufCapacity: 1, itemsAmount: 0, want: true},
		{bufCapacity: 1, itemsAmount: 1, want: false},
		{bufCapacity: 5, itemsAmount: 2, want: false},
		{bufCapacity: 3, itemsAmount: 8, want: false},
		{bufCapacity: 5, itemsAmount: 0, want: true},
		{bufCapacity: 5, itemsAmount: 1, want: false},
		{bufCapacity: 5, itemsAmount: 6, want: false},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("items: %d, cap: %d", tc.bufCapacity, tc.itemsAmount)
		t.Run(name, func(t *testing.T) {
			buffer, err := New[int](tc.bufCapacity)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < tc.itemsAmount; i++ {
				buffer.Push(42)
			}

			got := buffer.IsEmpty()
			if got != tc.want {
				t.Errorf("want: %t, got %t", tc.want, got)
			}

			for {
				_, ok := buffer.Pop()
				if !ok {
					break
				}
			}

			if !buffer.IsEmpty() {
				t.Errorf("buffer should be empty: got size:%d", buffer.Size())
			}
		})
	}
}

func TestRingBufferPushMultiThreading(t *testing.T) {
	generateTestItems := func(itemCount int) []int {
		items := make([]int, itemCount)
		for i := 0; i < itemCount; i++ {
			items[i] = rand.Intn(100)
		}
		return items
	}

	testCases := []struct {
		bufCapacity   int
		testItemCount int
		gorAmount     int
	}{
		{bufCapacity: 100, testItemCount: 100, gorAmount: 4},
		{bufCapacity: 22, testItemCount: 55, gorAmount: 11},
		{bufCapacity: 101, testItemCount: 100, gorAmount: 1},
		{bufCapacity: 3, testItemCount: 100, gorAmount: 10},
		{bufCapacity: 100, testItemCount: 55, gorAmount: 5},
		{bufCapacity: 10, testItemCount: 100, gorAmount: 7},
		{bufCapacity: 1000, testItemCount: 987, gorAmount: 41},
		{bufCapacity: 75, testItemCount: 50, gorAmount: 2},
	}

	for _, tc := range testCases {
		fmtStr := "cap: %d, items: %d, goroutines: %d"
		name := fmt.Sprintf(fmtStr, tc.bufCapacity, tc.testItemCount, tc.gorAmount)
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			testItems := generateTestItems(tc.testItemCount)
			buffer, err := New[int](tc.bufCapacity)
			if err != nil {
				t.Fatal(err)
			}

			itemChan := make(chan int, tc.gorAmount)
			for i := 0; i < tc.gorAmount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for item := range itemChan {
						buffer.Push(item)
					}
				}()
			}

			for i := 0; i < tc.testItemCount; i++ {
				itemChan <- testItems[i]
			}
			close(itemChan)

			wg.Wait()

			if buffer.Size() != min(tc.testItemCount, cap(buffer.data)) {
				t.Errorf("items in buffer: want %d, got %d", tc.testItemCount, buffer.Size())
			}
		})
	}
}

func TestRingBufferNew(t *testing.T) {
	t.Run("capacity 0", func(t *testing.T) {
		capacity := 0
		_, err := New[struct{}](capacity)
		if !errors.Is(err, ErrInvalidBuffCap) {
			t.Errorf("want error: %s, got error: %s", ErrInvalidBuffCap, err)
		}
	})

	t.Run("negative capacity", func(t *testing.T) {
		capacity := -1
		_, err := New[struct{}](capacity)
		if !errors.Is(err, ErrInvalidBuffCap) {
			t.Errorf("want error: %s, got error: %s", ErrInvalidBuffCap, err)
		}
	})

	t.Run("positive capacity", func(t *testing.T) {
		capacity := 1
		_, err := New[struct{}](capacity)
		if !errors.Is(err, nil) {
			t.Errorf("want error: %s, got error: %s", ErrInvalidBuffCap, err)
		}
	})
}

func TestRingBufferContainsAllItems(t *testing.T) {
	gorAmount := 99
	itemCount := 12345
	buffer, err := New[int](itemCount)
	if err != nil {
		t.Fatal(err)
	}

	itemChan := make(chan int)
	testItems := make([]int, itemCount)

	for i := 0; i < itemCount; i++ {
		testItems[i] = i
	}

	var wg sync.WaitGroup
	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for item := range itemChan {
				buffer.Push(item)
			}
		}(i)
	}

	for i := 0; i < itemCount; i++ {
		itemChan <- testItems[i]
	}
	close(itemChan)

	wg.Wait()
	sort.Ints(buffer.data)
	if !reflect.DeepEqual(buffer.data, testItems) {
		t.Errorf("items in buffer do not match test data")
	}
}

func TestRingBufferGetString(t *testing.T) {
	buffer, err := New[string](3)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("empty buffer", func(t *testing.T) {
		got, ok := buffer.Get()
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
		if ok != false {
			t.Errorf("expected ok: false, got ok: %t", ok)
		}
	})

	t.Run("buffer with items", func(t *testing.T) {
		buffer.Push("apple")
		buffer.Push("orange")
		want := "apple"
		got, ok := buffer.Get()
		if got != want {
			t.Errorf("expected %s, got %q", want, got)
		}
		if ok != true {
			t.Errorf("expected ok: true, got ok: %t", ok)
		}
	})
}

func TestRingBufferClear(t *testing.T) {
	itemCount := 100
	buffer, err := New[int](itemCount)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < itemCount; i++ {
		buffer.Push(42)
	}
	if buffer.IsEmpty() {
		t.Errorf("non-empty buffer expected")
	}

	buffer.Clear()
	if !buffer.IsEmpty() {
		t.Errorf("empty buffer expected")
	}

	if buffer.Size() != 0 {
		t.Errorf("buffer size: want 0, got %d", buffer.Size())
	}

	got, ok := buffer.Get()
	if got != 0 {
		t.Errorf("expected empty buffer, got %d", got)
	}
	if ok != false {
		t.Errorf("expected ok: false, got ok: %t", ok)
	}

	got, ok = buffer.Pop()
	if got != 0 {
		t.Errorf("expected empty buffer, got %d", got)
	}
	if ok != false {
		t.Errorf("expected ok: false, got ok: %t", ok)
	}
}

func TestRingBufferDeepClear(t *testing.T) {
	itemCount := 100
	buffer, err := New[int](itemCount)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < itemCount; i++ {
		buffer.Push(42)
	}
	if buffer.IsEmpty() {
		t.Errorf("non-empty buffer expected")
	}

	buffer.DeepClear()
	if !buffer.IsEmpty() {
		t.Errorf("empty buffer expected")
	}

	if buffer.Size() != 0 {
		t.Errorf("buffer size: want 0, got %d", buffer.Size())
	}

	got, ok := buffer.Get()
	if got != 0 {
		t.Errorf("expected empty buffer, got %d", got)
	}
	if ok != false {
		t.Errorf("expected ok: false, got ok: %t", ok)
	}

	got, ok = buffer.Pop()
	if got != 0 {
		t.Errorf("expected empty buffer, got %d", got)
	}
	if ok != false {
		t.Errorf("expected ok: false, got ok: %t", ok)
	}
}

func TestRingBufferReuseAfterClear(t *testing.T) {
	itemCount := 50
	buffer, err := New[int](itemCount)
	if err != nil {
		t.Fatal(err)
	}
	// Initially fill the buffer, just to clear it then.
	for i := 0; i < itemCount; i++ {
		buffer.Push(42)
	}
	buffer.Clear()

	testNumbers := randomNumbers(itemCount, -100, 100)
	for _, num := range testNumbers {
		buffer.Push(num)
	}
	if !buffer.IsFull() {
		t.Errorf("expected full buffer")
	}
	gotSize := buffer.Size()
	if gotSize != len(testNumbers) {
		t.Errorf("buffer size: got %d, want %d", gotSize, len(testNumbers))
	}
	if !reflect.DeepEqual(buffer.data, testNumbers) {
		t.Errorf("buffer data:\n got %v, \nwant %v", buffer.data, testNumbers)
	}

	idx := 0
	for !buffer.IsEmpty() {
		num, _ := buffer.Pop()
		if num != testNumbers[idx] {
			t.Errorf("Pop() item: got %d, want %d", num, testNumbers[idx])
		}
		idx++
	}
}

func TestRingBufferIsFull(t *testing.T) {
	testCases := []struct {
		bufferCap int
		itemCount int
		want      bool
	}{
		{bufferCap: 1, itemCount: 0, want: false},
		{bufferCap: 1, itemCount: 1, want: true},
		{bufferCap: 10, itemCount: 9, want: false},
		{bufferCap: 100, itemCount: 100, want: true},
		{bufferCap: 120, itemCount: 0, want: false},
		{bufferCap: 77, itemCount: 290, want: true},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("cap: %d, items: %d", tc.bufferCap, tc.itemCount)
		t.Run(name, func(t *testing.T) {
			buffer, err := New[int](tc.bufferCap)
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; i < tc.itemCount; i++ {
				buffer.Push(1)
			}

			got := buffer.IsFull()
			if got != tc.want {
				t.Errorf("want: %t, got: %t", tc.want, got)
			}
		})
	}
}

func TestRingBufferSize(t *testing.T) {
	testCases := []struct {
		name      string
		bufferCap int
		itemCount int
		wantSize  int
	}{
		{bufferCap: 1, itemCount: 0, wantSize: 0},
		{bufferCap: 1, itemCount: 1, wantSize: 1},
		{bufferCap: 10, itemCount: 9, wantSize: 9},
		{bufferCap: 100, itemCount: 100, wantSize: 100},
		{bufferCap: 120, itemCount: 0, wantSize: 0},
		{bufferCap: 77, itemCount: 290, wantSize: 77},
		{bufferCap: 33, itemCount: 99, wantSize: 33},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("cap: %d, items: %d", tc.bufferCap, tc.itemCount)
		t.Run(name, func(t *testing.T) {
			buffer, err := New[bool](tc.bufferCap)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < tc.itemCount; i++ {
				buffer.Push(true)
			}

			if buffer.Size() != tc.wantSize {
				t.Errorf("size: want %d, got %d", tc.wantSize, buffer.Size())
			}
		})
	}
}

func TestRingBufferCapacity(t *testing.T) {
	testCases := []struct {
		name      string
		bufferCap int
	}{
		{name: "fixed capacity", bufferCap: 4},
		{name: "random capacity", bufferCap: rand.Intn(1000) + 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer, err := New[int](tc.bufferCap)
			if err != nil {
				t.Fatal(err)
			}

			if buffer.Capacity() != tc.bufferCap {
				t.Errorf("buffer capacity: want %d, got %d", tc.bufferCap, buffer.Capacity())
			}
		})
	}
}

func TestRingBufferDetectDataRace(t *testing.T) {
	bufferCap := 500
	gorAmount := 100
	opCount := 10_000

	buffer, err := New[string](bufferCap)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				item := fmt.Sprintf("item %d from goroutine %d", j, i)
				buffer.Push(item)
			}
		}(i)
	}

	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				buffer.Size()
			}
		}()
	}

	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				buffer.Capacity()
			}
		}()
	}

	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				buffer.Get()
			}
		}()
	}

	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				buffer.Pop()
			}
		}()
	}

	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				buffer.IsEmpty()
			}
		}()
	}

	for i := 0; i < gorAmount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opCount; j++ {
				buffer.IsFull()
			}
		}()
	}

	wg.Wait()
}

// randomNumbers returns a slice of size random integers
// between min and max (exclusive).
func randomNumbers(size, min, max int) []int {
	numbers := make([]int, size)

	for i := 0; i < size; i++ {
		numbers[i] = rand.Intn(max-min) + min
	}

	return numbers
}
