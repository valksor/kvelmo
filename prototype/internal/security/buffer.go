package security

// maxOutputSize defines the maximum output size for scanner command output.
// This limit prevents memory exhaustion from maliciously large outputs.
const maxOutputSize = 10 * 1024 * 1024 // 10MB

// limitedBuffer is a buffer with a size limit to prevent memory exhaustion
// from external command output. It implements io.Writer.
//
// When the limit is reached, additional writes are silently discarded.
// This is intentional - we want the command to complete rather than error
// on large output, but we cap memory usage.
type limitedBuffer struct {
	buf   []byte
	limit int
}

// newLimitedBuffer creates a new limitedBuffer with the specified size limit.
func newLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{
		limit: limit,
	}
}

// Write appends data to the buffer up to the limit.
// Once the limit is reached, additional writes are silently discarded.
// The full length of p is always returned to satisfy io.Writer contract.
func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - len(b.buf)
	if remaining <= 0 {
		return len(p), nil // Silent discard when limit reached
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	b.buf = append(b.buf, p...)

	return len(p), nil
}

// Bytes returns the accumulated bytes.
func (b *limitedBuffer) Bytes() []byte {
	return b.buf
}

// String returns the accumulated bytes as a string.
func (b *limitedBuffer) String() string {
	return string(b.buf)
}

// Len returns the current length of the buffer.
func (b *limitedBuffer) Len() int {
	return len(b.buf)
}

// Reset clears the buffer, retaining the underlying storage.
func (b *limitedBuffer) Reset() {
	b.buf = b.buf[:0]
}
