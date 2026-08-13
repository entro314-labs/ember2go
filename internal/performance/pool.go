package performance

import (
	"bytes"
	"sync"
)

// BufferPool provides reusable byte buffers to reduce GC pressure
var BufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 32*1024)) // 32KB initial capacity
	},
}

// GetBuffer gets a buffer from the pool
func GetBuffer() *bytes.Buffer {
	return BufferPool.Get().(*bytes.Buffer)
}

// PutBuffer returns a buffer to the pool after resetting it
func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	BufferPool.Put(buf)
}

// StringBuilderPool provides reusable string builders
var StringBuilderPool = sync.Pool{
	New: func() interface{} {
		var sb bytes.Buffer
		sb.Grow(1024) // Pre-allocate 1KB
		return &sb
	},
}

// GetStringBuilder gets a string builder from the pool
func GetStringBuilder() *bytes.Buffer {
	return StringBuilderPool.Get().(*bytes.Buffer)
}

// PutStringBuilder returns a string builder to the pool
func PutStringBuilder(sb *bytes.Buffer) {
	sb.Reset()
	StringBuilderPool.Put(sb)
}

// SlicePool provides reusable string slices for parsing operations
var SlicePool = sync.Pool{
	New: func() interface{} {
		return make([]string, 0, 64) // Pre-allocate for 64 strings
	},
}

// GetStringSlice gets a string slice from the pool
func GetStringSlice() []string {
	return SlicePool.Get().([]string)
}

// PutStringSlice returns a string slice to the pool
func PutStringSlice(slice []string) {
	// Clear the slice but keep capacity
	slice = slice[:0]
	SlicePool.Put(slice)
}
