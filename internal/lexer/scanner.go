package lexer

import (
	"io"
	"slices"
)

const defaultScannerBufferSize = 4096

type byteClass []byte

func (bc byteClass) contains(b byte) bool {
	return slices.Contains(bc, b)
}

func (bc byteClass) negate() byteClass {
	var negated []byte
	for i := range 256 {
		if !slices.Contains(bc, byte(i)) {
			negated = append(negated, byte(i))
		}
	}
	return negated
}

func byteClassRange(start, stop byte) byteClass {
	var cls []byte
	for b := start; b <= stop; b++ {
		cls = append(cls, b)
	}
	return cls
}

func byteClassChars(chars ...byte) byteClass {
	return []byte(chars)
}

func byteClassCombine(bClases ...byteClass) byteClass {
	var cls []byte
	for _, bClass := range bClases {
		cls = append(cls, bClass...)
	}
	slices.Sort(cls)
	return slices.Compact(cls)
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
		pos: buffSize,
		max: buffSize,
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

func (s *scanner) readOne() (byte, error) {
	if s.remaining() > 0 {
		b := s.buff[s.pos]
		s.pos++
		return b, nil
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

	return
}
