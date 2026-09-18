// Package mcp handles the JSON-RPC 2.0 line protocol MCP speaks over
// stdio: reading it off a stream and routing it by method.
package mcp

import (
	"bufio"
	"errors"
	"io"
)

const (
	// initialBufSize is the reader's fixed internal buffer.
	initialBufSize = 64 * 1024

	// MaxLineSize bounds a single line so it can't exhaust memory.
	MaxLineSize = 1024 * 1024
)

// ErrLineTooLong is returned when a line exceeds MaxLineSize. Scan has
// already drained and forwarded the rest of that line to rawOut by
// then, and stays usable for the next line.
var ErrLineTooLong = errors.New("mcp: line exceeds MaxLineSize")

// Scanner reads newline-delimited JSON-RPC messages from one stdio
// stream. One goroutine owns it for its whole life, so its buffer is
// just reused, never pooled.
type Scanner struct {
	r    *bufio.Reader
	line []byte
}

// NewScanner wraps r for line-delimited reading.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		r:    bufio.NewReaderSize(r, initialBufSize),
		line: make([]byte, 0, initialBufSize),
	}
}

// Scan reads the next line (result valid until the next call). Returns
// io.EOF at end of stream, or ErrLineTooLong for a line over MaxLineSize
// (drained and forwarded to rawOut, which may be nil).
func (s *Scanner) Scan(rawOut io.Writer) (line []byte, err error) {
	s.line = s.line[:0]
	for {
		chunk, rerr := s.r.ReadSlice('\n')
		if len(s.line)+len(chunk) > MaxLineSize {
			return nil, s.drainOverflow(chunk, rerr, rawOut)
		}
		s.line = append(s.line, chunk...)

		switch rerr {
		case nil:
			// ReadSlice includes the delimiter; the caller wants the
			// line without it.
			return s.line[:len(s.line)-1], nil
		case bufio.ErrBufferFull:
			continue // line spans more than one internal buffer fill
		case io.EOF:
			if len(s.line) == 0 {
				return nil, io.EOF
			}
			return s.line, nil // final line, no trailing newline
		default:
			return nil, rerr
		}
	}
}

// drainOverflow reads the rest of an oversized line up to its newline,
// writing every byte to rawOut without buffering the overflow.
func (s *Scanner) drainOverflow(chunk []byte, rerr error, rawOut io.Writer) error {
	if err := writeAll(rawOut, s.line); err != nil {
		return err
	}
	if err := writeAll(rawOut, chunk); err != nil {
		return err
	}
	for rerr == bufio.ErrBufferFull {
		chunk, rerr = s.r.ReadSlice('\n')
		if err := writeAll(rawOut, chunk); err != nil {
			return err
		}
	}
	if rerr != nil && rerr != io.EOF {
		return rerr
	}
	return ErrLineTooLong
}

func writeAll(w io.Writer, b []byte) error {
	if w == nil || len(b) == 0 {
		return nil
	}
	_, err := w.Write(b)
	return err
}
