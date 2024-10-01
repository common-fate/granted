// Package chromemsg implements the Chrome native messaging protocol:
// https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging
package chromemsg

import (
	"encoding/binary"
	"io"
)

type Server struct {
	Input  io.Reader
	Output io.Writer
}

func (s *Server) Read(p []byte) (int, error) {
	// From the Chrome developer docs:
	// "Each message is serialized using JSON, UTF-8 encoded and is preceded with 32-bit message length in native byte order."
	// we need to read the 32-bit message length value first.
	var b [4]byte
	_, err := io.ReadFull(s.Input, b[:])
	if err != nil {
		return 0, err
	}

	size := binary.LittleEndian.Uint32(b[:])
	r := io.LimitReader(s.Input, int64(size))

	return r.Read(p)
}

func (s *Server) Write(p []byte) (n int, err error) {
	// From the Chrome developer docs:
	// "Each message is serialized using JSON, UTF-8 encoded and is preceded with 32-bit message length in native byte order."
	// we need to read the 32-bit message length value first.

	header := []byte{0, 0, 0, 0} // 32-bit
	binary.LittleEndian.PutUint32(header, uint32(len(p)))

	n, err = s.Output.Write(header)
	if err != nil {
		return 0, err
	}

	n2, err := s.Output.Write(p)
	if err != nil {
		return 0, err
	}

	return n + n2, nil
}
