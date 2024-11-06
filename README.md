# Ring Buffer

This repository contains a thread-safe circular buffer implementation in Go. The circular buffer, also known as a ring buffer, is a data structure that uses a single, fixed-size buffer as if it were connected end-to-end. This structure lends itself easily to buffering data streams.

## Features

- **Thread-safe**: The buffer is designed to be used in concurrent environments.
- **Well tested**: Repository contains tests for all buffer methods.

## Usage

### Creating a Ring Buffer

```go
import ringBuf "github.com/H4RP3R/ring_buffer"

// Create circular buffer with capacity of 10 integers
buffer, err := ringBuf.New[int](10)
if err != nil {
    panic(err)
}
```

### Adding Elements

```go
// Push an element into the buffer
buffer.Push(42)
```

### Removing Elements

```go
// Pop an element from the buffer
item, ok := buffer.Pop()
if !ok {
    fmt.Println("Buffer is empty")
} else {
    fmt.Println("Popped:", item)
}
```

### Checking Buffer Status

```go
// Check if the buffer is empty
if buffer.IsEmpty() {
    fmt.Println("Buffer is empty")
}

// Check if the buffer is full
if buffer.IsFull() {
    fmt.Println("Buffer is full")
}

// Check the current size of the buffer
fmt.Println("Buffer size:", buffer.Size())

// Reset the buffer
buffer.Clear()

// Clear all elements from the buffer, writing zero values to all buffer cells
buffer.DeepClear()
```

### Getting Elements Without Removing

```go
// Get an element from the buffer without removing it
item, ok := buffer.Get()
if !ok {
    fmt.Println("Buffer is empty")
} else {
    fmt.Println("Current item:", item)
}
```

## API Reference

- `Push(item T)`: Adds an element to the buffer.
- `Pop() (item T, ok bool)`: Removes and returns an element from the beginning of the buffer.
- `IsEmpty() bool`: Checks if the buffer is empty.
- `Full() bool`: Checks if the buffer is full.
- `Size() int`: Returns the current size of the buffer.
- `Get() (item T, ok bool)`: Returns an element from the beginning of the buffer without removing it.
- `Clear()`: Resets the buffer to the initial state.
- `DeepClear()`: Clears the buffer, removing all elements by writing zero values to all buffer cells.

### New Function

- `New[T any](capacity int) (cb *cBuffer[T], err error)`: Creates a new circular buffer with the given capacity.

## Contributing

Contributions are welcome! If you find a bug or want to add a new feature, please open an issue or submit a pull request.
