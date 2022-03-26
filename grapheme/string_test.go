package grapheme

import (
	"io"
	"testing"

	"github.com/npillmayer/uax/internal/tracing"
)

func TestRuneReader(t *testing.T) {
	reader := &rr{
		input: "Hello World",
		pos:   0,
	}
	cnt := 0
	for {
		r, sz, err := reader.ReadRune()
		if err != nil {
			if err != io.EOF {
				t.Errorf(err.Error())
			} else {
				t.Logf(err.Error())
			}
			break
		}
		t.Logf("r = %#U, sz = %d", r, sz)
		cnt++
	}
	if cnt != 11 {
		t.Errorf("expected to read 11, have %d", cnt)
	}
}

func TestString(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	input := "Hello World"
	s := StringFromString(input)
	if s == nil {
		t.Fatalf("resulting grapheme string should not be nil")
	}
	t.Logf("breaks at %v", s.(*shortString).breaks)
	x := s.Nth(2)
	t.Logf("s.Nth(2) = %#U", x[0])
	if x != "l" {
		t.Errorf("expected s.Nth(2) to be 'l', is %#v", x)
	}
	l := s.Len()
	if l != 11 {
		t.Errorf("expected s.Len() to be 11, is %d", s.Len())
	}
}

func TestChineseString(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	input := "世界"
	s := StringFromString(input)
	if s == nil {
		t.Fatalf("resulting grapheme string should not be nil")
	}
	l := s.Len()
	if l != 2 {
		t.Errorf("expected \"%s\".Len() to be 2, is %d", input, s.Len())
	}
	x := s.Nth(1)
	t.Logf("s.Nth(1) = %s", x)
	t.Logf("number of bytes for 2nd grapheme: %d", len(s.Nth(1))) // => 3
	if x != "界" {
		t.Errorf("expected s.Nth(1) to be '界', is %s", x)
	}
}
