package lexer

import (
	"cmp"
	"io"
	"strings"
	"testing"
)

func TestScanner_peekOne(t *testing.T) {
	s := newScanner(strings.NewReader("#\n"), 0)
	b, err := s.peekOne()
	if err != nil {
		t.Fatal("unexpected error ", err)
	}
	if b != '#' {
		t.Fatalf("wanted '#' got '%c'", b)
	}
}

func TestScanner_readOne(t *testing.T) {
	s := newScanner(strings.NewReader("#\n"), 0)
	b, err := s.readOne()
	if err != nil {
		t.Fatal("unexpected error ", err)
	}
	if b != '#' {
		t.Fatalf("wanted '#' got '%c'", b)
	}
}

func TestScanner_basicParsing(t *testing.T) {
	bcNotSlash := byteClassChars('/').negate()
	if l := len(bcNotSlash.String()); l != 255 {
		t.Fatalf("expecting everything except slash to be 255 in size, not %d", l)
	}
	if bcNotSlash.contains('/') {
		t.Fatal("expected byteClass without / to not contains it")
	}

	r := strings.NewReader("foo/*maybe*coment*/bar")
	s := newScanner(r, 16)

	gotFoo, haveMore, err := s.readBytesInClass(bcNotSlash)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	if haveMore {
		t.Fatal("there should be still data in buffer")
	}
	if string(gotFoo) != "foo" {
		t.Fatalf("expected foo, got %q", gotFoo)
	}

	// simulating to parse "/*"
	b1, err1 := s.readOne()
	b2, err2 := s.readOne()
	if err := cmp.Or(err1, err2); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if b1 != '/' || b2 != '*' {
		t.Fatalf(`expected "/*" but got "%c%c"`, b1, b2)
	}
}

func TestScanner_fillingNotEmptyBuffer(t *testing.T) {
	hexedecimalByteClass := byteClassCombine(
		byteClassRange('0', '9'),
		byteClassRange('A', 'F'),
		byteClassRange('a', 'f'),
	)
	if s := hexedecimalByteClass.String(); len(s) != 22 {
		t.Fatalf("expected all hexedecimal, got %q", s)
	}

	r := strings.NewReader("09afAF-")
	s := newScanner(r, 6)

	data, isPrefix, err := s.readBytesInClass(hexedecimalByteClass)
	if err != nil {
		t.Fatal("unexpected error ", err)
	}
	if string(data) != "09afAF" || !isPrefix {
		t.Fatalf(
			"last read should have clear all buffer, but read %q and and left %q",
			string(data), string(s.buff),
		)
	}
	// all buffer read, isPrefix == true, so we should check for continuation
	data, isPrefix, err = s.readBytesInClass(hexedecimalByteClass)
	if err != nil {
		t.Fatal("unexpected error ", err)
	}
	if len(data) != 0 || isPrefix {
		t.Fatalf("there should be no data in continuation, but got %q", string(data))
	}

	b, err := s.peekOne()
	if err != nil {
		t.Fatal("unexpected error ", err)
	}
	if b != '-' {
		t.Fatalf("expected '-' but got '%c'", b)
	}

	// Following code is to detect required panic
	havePanicked := false
	defer func(b *bool) {
		t.Helper()
		if !*b {
			t.Fatal("expected fillBuffer on not empty to panic")
		}
	}(&havePanicked)
	defer func(b *bool) {
		p := recover()
		if p != nil {
			t.Logf("recovered %v", p)
			*b = true
		}
	}(&havePanicked)

	// now lets assume cleaver user code wanted to discord '-' char
	// and calling fillBuffer directly
	// code should not silently discard buffer data
	s.fillBuffer()
}

func TestScanner_onEOF(t *testing.T) {
	r := strings.NewReader("")
	s := newScanner(r, 128)

	if b, err := s.peekOne(); err != io.EOF {
		t.Fatalf("expected io.EOF but got '%c' %v", b, err)
	}

	if b, err := s.readOne(); err != io.EOF {
		t.Fatalf("expected io.EOF but got '%c' %v", b, err)
	}

	if data, _, err := s.readBytesInClass(byteClassEmpty()); err != io.EOF {
		t.Fatalf("expected io.EOF but got %q %v", data, err)
	}
}

type singleByteReader byte

var _ io.Reader = singleByteReader('#')

func (r singleByteReader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, io.ErrShortBuffer
	}
	dst[0] = byte(r)
	return 1, io.EOF
}

func TestScanner_handlingShortReadWithEof(t *testing.T) {
	s := newScanner(singleByteReader('$'), 0)
	b, err := s.readOne()
	if err != nil {
		t.Fatal("reader returns err with every longer read, should be moved to second call ", err)
	}
	if b != '$' {
		t.Fatalf("expected '$' got '%c'", b)
	}

	// second read, we should get io.EOF
	b, err = s.readOne()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %c and %v", b, err)
	}
}
