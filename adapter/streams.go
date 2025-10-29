package adapter

import (
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
	total := len(p)
	writeSize := total

	for writeSize > 0 {
		// Check available space
		for {
			size, err, iserr := s.inner.CheckWrite().Result()
			if iserr {
				switch err.Tag() {
				case 0:
					errDetail := err.LastOperationFailed()
					defer errDetail.ResourceDrop()
					return total - writeSize, errors.New(errDetail.ToDebugString())
				case 1:
					return total - writeSize, io.EOF
				}
			}
			if size > 0 {
				if size < uint64(writeSize) {
					writeSize = int(size)
				}
				break
			}
			s.pollable.Block()
		}

		// Write in chunks
		_, err, iserr := s.inner.Write(cm.ToList(p[:writeSize])).Result()
		if iserr {
			switch err.Tag() {
			case 0:
				errDetail := err.LastOperationFailed()
				defer errDetail.ResourceDrop()
				return total - writeSize, errors.New(errDetail.ToDebugString())
			case 1:
				return total - writeSize, io.EOF
			}
		}
		p = p[writeSize:]
		writeSize = len(p)
	}

	return total, nil
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
	inner drivertypes.InputStream
}

func (s *InputStream) Read(p []byte) (int, error) {
	data, err, iserr := s.inner.BlockingRead(min(uint64(len(p)), 1<<20)).Result()
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

func (s *InputStream) Close() error {
	s.inner.ResourceDrop()
	return nil
}

func NewInputStream(inner drivertypes.InputStream) InputStream {
	return InputStream{
		inner: inner,
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

func (us *UploadRequest) Peek(offset uint64, length uint64) (io.ReadCloser, error) {
	stream, err, iserr := us.Content.Peek(offset, length).Result()
	if iserr {
		return nil, errors.New(err)
	}
	is := NewInputStream(stream)
	return &is, nil
}

func (us *UploadRequest) Chunks(chunkSize uint32) (chunkCount uint32, e error) {
	count, err, iserr := us.Content.Chunks(chunkSize).Result()
	if iserr {
		return 0, errors.New(err)
	}
	return count, nil
}

func (us *UploadRequest) NextChunk() (io.ReadCloser, error) {
	stream, err, iserr := us.Content.NextChunk().Result()
	if iserr {
		return nil, errors.New(err)
	}
	is := NewInputStream(stream)
	return &is, nil
}

func (us *UploadRequest) ChunkReset(chunk io.ReadCloser) error {
	is, ok := chunk.(*InputStream)
	if !ok {
		return errors.New("invalid chunk type")
	}
	_, err, iserr := us.Content.ChunkReset(is.inner).Result()
	if iserr {
		return errors.New(err)
	}
	return nil
}
