package adapter

import (
	"errors"
	"io"

	drivertypes "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/types"
	"github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/wasi/io/poll"
	"go.bytecodealliance.org/cm"
)

// Helper functions for min/max
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type OutputStream struct {
	inner       drivertypes.OutputStream
	pollable    poll.Pollable
	writeBuffer []byte // Reusable buffer for batch writes
	bufferSize  int    // Current buffer usage
}

func NewOutputStream(inner drivertypes.OutputStream) OutputStream {
	pollable := inner.Subscribe()
	return OutputStream{
		inner:       inner,
		pollable:    pollable,
		writeBuffer: make([]byte, 0, 32768), // 32KB initial capacity
		bufferSize:  0,
	}
}

func (s *OutputStream) Write(p []byte) (int, error) {
	totalWritten := 0
	remaining := p

	for len(remaining) > 0 {
		// Check available space
		var checkSize uint64
		for {
			size, err, iserr := s.inner.CheckWrite().Result()
			if iserr {
				switch err.Tag() {
				case 0:
					errDetail := err.LastOperationFailed()
					defer errDetail.ResourceDrop()
					return totalWritten, errors.New(errDetail.ToDebugString())
				case 1:
					return totalWritten, io.EOF
				}
			}
			if size > 0 {
				checkSize = size
				break
			}
			s.pollable.Block()
		}

		// Write in chunks
		writeSize := min(checkSize, uint64(len(remaining)))
		s.inner.Write(cm.ToList(remaining[:writeSize]))
		totalWritten += int(writeSize)
		remaining = remaining[writeSize:]
	}

	return totalWritten, nil
}

func (s *OutputStream) Close() error {
	defer s.inner.ResourceDrop()
	defer s.pollable.ResourceDrop()

	_, err, iserr := s.inner.BlockingFlush().Result()
	if iserr {
		switch err.Tag() {
		case 0:
			err := err.LastOperationFailed()
			defer err.ResourceDrop()
			return errors.New(err.ToDebugString())
		case 1:
			return io.EOF
		}
	}

	return nil
}

type InputStream struct {
	inner      drivertypes.InputStream
	readBuffer []byte // Reusable buffer for reads
	bufPos     int    // Current position in buffer
	bufLen     int    // Valid data length in buffer
}

func (s *InputStream) Read(p []byte) (int, error) {
	// If we have buffered data, use it first
	if s.bufPos < s.bufLen {
		n := copy(p, s.readBuffer[s.bufPos:s.bufLen])
		s.bufPos += n
		if s.bufPos >= s.bufLen {
			s.bufPos = 0
			s.bufLen = 0
		}
		return n, nil
	}

	// Read chunk size: larger of requested or 512KB for efficiency
	readSize := uint64(max(len(p), 524288))
	if readSize > 1<<20 { // Cap at 1MB
		readSize = 1 << 20
	}

	data, err, iserr := s.inner.BlockingRead(readSize).Result()
	if iserr {
		switch err.Tag() {
		case 0:
			errDetail := err.LastOperationFailed()
			defer errDetail.ResourceDrop()
			return 0, errors.New(errDetail.ToDebugString())
		case 1:
			return 0, io.EOF
		}
	}

	dataSlice := data.Slice()
	n := copy(p, dataSlice)
	
	// Buffer remaining data if any
	if n < len(dataSlice) {
		remaining := len(dataSlice) - n
		if cap(s.readBuffer) < remaining {
			s.readBuffer = make([]byte, remaining)
		} else {
			s.readBuffer = s.readBuffer[:remaining]
		}
		s.bufLen = copy(s.readBuffer, dataSlice[n:])
		s.bufPos = 0
	}

	return n, nil
}

func (s *InputStream) Close() error {
	s.inner.ResourceDrop()
	return nil
}

func NewInputStream(inner drivertypes.InputStream) InputStream {
	return InputStream{
		inner:      inner,
		readBuffer: make([]byte, 0, 524288), // 512KB initial capacity
		bufPos:     0,
		bufLen:     0,
	}
}

type UploadRequest struct {
	drivertypes.UploadRequest
}

func (us *UploadRequest) GetHash(hashs []drivertypes.HashAlg) ([]drivertypes.HashInfo, error) {
	infos, err, iserr := us.Content.GetHasher(cm.ToList(hashs)).Result()
	if iserr {
		return nil, errors.New(err)
	}
	return infos.Slice(), nil
}

func (us *UploadRequest) UpdateProgress(progress float64) {
	us.Content.UpdateProgress(progress)
}

func (us *UploadRequest) Streams() (io.ReadCloser, error) {
	stream, err, iserr := us.Content.Streams().Result()
	if iserr {
		return nil, errors.New(err)
	}
	is := NewInputStream(stream)
	return &is, nil
}

func (us *UploadRequest) Peek(offset uint64, start uint64) (io.ReadCloser, error) {
	stream, err, iserr := us.Content.Peek(offset, start).Result()
	if iserr {
		return nil, errors.New(err)
	}
	is := NewInputStream(stream)
	return &is, nil
}

func (us *UploadRequest) Range(offset uint64, start uint64) (io.ReadCloser, error) {
	stream, err, iserr := us.Content.Range(offset, start).Result()
	if iserr {
		return nil, errors.New(err)
	}
	is := NewInputStream(stream)
	return &is, nil
}
