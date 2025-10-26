package adapter

import (
	"bufio"
	"errors"
	"io"

	drivertypes "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/types"
	"github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/wasi/io/poll"
	"go.bytecodealliance.org/cm"
)

type OutputStream struct {
	inner    drivertypes.OutputStream
	pollable poll.Pollable
}

func NewOutputStream(inner drivertypes.OutputStream) OutputStream {
	pollable := inner.Subscribe()
	return OutputStream{
		inner:    inner,
		pollable: pollable,
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
	inner  drivertypes.InputStream
	reader *bufio.Reader
}

func (s *InputStream) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *InputStream) Close() error {
	s.inner.ResourceDrop()
	return nil
}

// baseInputStream wraps the WASI InputStream for use with bufio.Reader
type baseInputStream struct {
	inner drivertypes.InputStream
}

func (r *baseInputStream) Read(p []byte) (int, error) {
	// Read with optimal chunk size
	readSize := uint64(len(p))
	if readSize < 64*1024 {
		readSize = 64 * 1024 // Minimum 64KB reads
	}
	if readSize > 1<<20 {
		readSize = 1 << 20 // Cap at 1MB
	}

	data, err, iserr := r.inner.BlockingRead(readSize).Result()
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

	return copy(p, data.Slice()), nil
}

func NewInputStream(inner drivertypes.InputStream) InputStream {
	reader := &baseInputStream{inner: inner}
	return InputStream{
		inner:  inner,
		reader: bufio.NewReaderSize(reader, 512*1024), // 512KB buffer
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
