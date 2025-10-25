package adapter

import (
	"io"
	"sync"
)

const (
	// Optimal buffer sizes for different scenarios
	SmallFileThreshold  = 1 << 20      // 1MB
	MediumFileThreshold = 10 << 20     // 10MB
	SmallBufferSize     = 32 << 10     // 32KB
	MediumBufferSize    = 128 << 10    // 128KB
	LargeBufferSize     = 512 << 10    // 512KB
)

// Buffer pool for reusing large buffers
var copyBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, LargeBufferSize)
		return &buf
	},
}

// BufferedCopy performs optimized copying with adaptive buffering
// It selects buffer size based on estimated data size and reuses buffers
func BufferedCopy(dst io.Writer, src io.Reader, estimatedSize int64) (written int64, err error) {
	var buf []byte

	// Select buffer size based on estimated transfer size
	if estimatedSize > 0 && estimatedSize < SmallFileThreshold {
		buf = make([]byte, SmallBufferSize)
	} else if estimatedSize > 0 && estimatedSize < MediumFileThreshold {
		buf = make([]byte, MediumBufferSize)
	} else {
		// Use pooled buffer for large files
		bufPtr := copyBufferPool.Get().(*[]byte)
		buf = *bufPtr
		defer copyBufferPool.Put(bufPtr)
	}

	return io.CopyBuffer(dst, src, buf)
}

// ChunkedCopy performs chunked transfer with progress callback
func ChunkedCopy(dst io.Writer, src io.Reader, chunkSize int, progressFn func(int64)) (written int64, err error) {
	if chunkSize <= 0 {
		chunkSize = LargeBufferSize
	}

	buf := make([]byte, chunkSize)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
			if progressFn != nil {
				progressFn(written)
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
