package lexer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

const defaultScannerBufferSize = 4096

type byteClass [256]bool

var _ fmt.Stringer = byteClassEmpty()

func (bc byteClass) String() string {
	var builder strings.Builder
	for i, b := range bc {
		if b {
			builder.WriteByte(byte(i))
		}
	}
	return builder.String()
}

func (bc byteClass) contains(b byte) bool {
	return bc[b]
}

func (bc byteClass) negate() byteClass {
	var negated byteClass
	for i := range bc {
		negated[i] = !bc[i]
	}
	return negated
}

func byteClassRange(start, stop byte) byteClass {
	var cls byteClass
	for b := start; b <= stop; b++ {
		cls[b] = true
	}
	return cls
}

func byteClassChars(chars ...byte) byteClass {
	var cls byteClass
	for _, b := range chars {
		cls[b] = true
	}
	return cls
}

func byteClassCombine(bClases ...byteClass) byteClass {
	var cls byteClass
	for i := range cls {
		for _, bClass := range bClases {
			if bClass.contains(byte(i)) {
				cls[i] = true
			}
		}
	}
	return cls
}

func byteClassEmpty() byteClass {
	return *new(byteClass)
}

// standard library buffered readers and scanners
// doesn't support reading up to any byte, but single delim.
// If we have to wrap anyway, let's wrap basic io.Reader
type scanner struct {
	reader io.Reader

	buff []byte

	pos, max int

	// acording to https://pkg.go.dev/io#Reader
	// reader can return n>0 and error in Read()
	// we store it for subsequent calls
	err error

	// counting lines and columns further down is such a pain
	line, column int
}

func newScanner(r io.Reader, buffSize int) *scanner {
	if buffSize <= 0 {
		buffSize = defaultScannerBufferSize
	}
	return &scanner{
		reader: r,
		buff:   make([]byte, buffSize),
		// starting with buffSize initial position
		// to force read in first peek/read instead of constructor
		pos:    buffSize,
		max:    buffSize,
		line:   1,
		column: 1,
	}
}

func (s *scanner) remaining() int {
	return s.max - s.pos
}

func (s *scanner) fillBuffer() error {
	if err := s.err; err != nil {
		s.err = nil
		return err
	}

	if r := s.remaining(); r > 0 {
		// This edge case shoud not happen, without calling this method directly.
		// I prefear not to implement this edge case.
		// It is not needed, and would bring buffor compaction and io.EOF handling.
		// Silently overwriting buffer data wouldalso be error prone.
		panic("filling not empty buffer")
	}

	n, err := s.reader.Read(s.buff)
	if err != nil && n == 0 {
		return err
	}
	if err != nil {
		s.err = err
	}

	s.pos = 0
	s.max = n

	return nil
}

func (s *scanner) popOneFromBuffer() byte {
	// only operation on buffer so no error can ocur
	// sometimes peeked byte can be discarded
	// without readOne() and err check
	if s.remaining() == 0 {
		panic("discard byte on empty buffer")
	}
	b := s.buff[s.pos]

	if b == '\n' {
		s.line++
		s.column = 1
	} else {
		s.column++
	}

	s.pos++
	return b
}

func (s *scanner) readOne() (byte, error) {
	if s.remaining() > 0 {
		return s.popOneFromBuffer(), nil
	}

	err := s.fillBuffer()
	if err != nil {
		return 0, err
	}

	return s.readOne()
}

func (s *scanner) peekOne() (byte, error) {
	if s.remaining() > 0 {
		return s.buff[s.pos], nil
	}

	err := s.fillBuffer()
	if err != nil {
		return 0, err
	}

	return s.peekOne()
}

// Reads all bytes curently in buffer that are in specified byteClass.
// Returns read bytes in slice, thats **only valid up to next call**.
// There is chance that big portion of input data will be discarded,
// for example in #if/#ifde/#ifndef, and harder to use api was chosen,
// that will not allocate for throw away data.
func (s *scanner) readBytesInClass(cls byteClass) (data []byte, isPrefix bool, err error) {
	if s.remaining() == 0 {
		err = s.fillBuffer()
		if err != nil {
			return nil, false, err
		}
	}

	end := s.pos
	for i := s.pos; i < s.max; i++ {
		if !cls.contains(s.buff[i]) {
			break
		}
		end = i + 1
	}

	if end == s.pos {
		// no datamatched, nothing to read, no continuation and no error
		return nil, false, nil
	}

	data = s.buff[s.pos:end]
	s.pos = end
	isPrefix = end == s.max

	s.advanceLineColumn(data)
	return
}

func (s *scanner) advanceLineColumn(data []byte) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		s.line++
		s.column = 1
		s.advanceLineColumn(data[i+1:])
	} else {
		s.column += len(data)
	}
}
