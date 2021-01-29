package grapheme

import (
	"fmt"
	"io"
	"math"
	"unicode/utf8"

	"github.com/npillmayer/uax/segment"
)

// String is a type to represent a graheme string, i.e. a sequence of
// “user perceived characters” as defined by Unicode.
// A grapheme string is a read-only data structure.
//
// Finding graphemes from a string (or array of bytes) is an operation with
// runtime complexiy O(N). Clients should not convert large texts into grapheme
// strings in one go, but rather operate on manageable fragments.
//
type String interface {
	Nth(int) string // return nth grapheme
	Len() int       // length of string in units of user perceived characters
}

// MaxByteLen is the maximum byte count a grapheme string may consist of.
const MaxByteLen int = 32766

// StringFromString creates a grapheme string from a Go string.
// As grapheme strings are not meant to be created for large amounts of text, but
// rather for manageable segments, s is not allowed to exceed s^16-1 = 32766 bytes.
//
// StringFromString will panic if a larger input string is given.
//
// StringFromString will trim the input Go string to valid Unicode code point (rune)
// boundaries. If s does not contain any legal runes, the resulting grapheme string
// may be of length 0 even if the input string is not.
//
func StringFromString(s string) String {
	if len(s) < math.MaxUint8 {
		return makeShortString(s)
	} else if len(s) < math.MaxUint16 {
		return makeMidString(s)
	}
	panic(fmt.Sprintf("grapheme.String may not be built from more than %d bytes, have %d",
		MaxByteLen, len(s)))
}

// StringFromBytes creates a grapheme string from an array of bytes. As grapheme
// strings are a read-only data structure, StringFromBytes will create a private copy
// of the input.
//
// As grapheme strings are not meant to be created for large amounts of text, but
// rather for manageable segments, b is not allowed to exceed s^16-1 = 32766 bytes.
//
// StringFromBytes will panic if a larger input slice is given.
//
// StringFromBytes will trim the input to valid Unicode code point (rune)
// boundaries. If b does not contain any legal runes, the resulting grapheme string
// may be of length 0 even if the input slice is not.
//
//
func StringFromBytes(b []byte) String {
	return StringFromString(string(b))
}

// --- Short version ---------------------------------------------------------

type shortString struct {
	content string
	breaks  []uint8
}

func makeShortString(s string) String {
	gstr := &shortString{content: s}
	breaker := prepareBreaking(s)
	if breaker == nil {
		return gstr
	}
	gstr.breaks = make([]uint8, 1, len(s)/4)
	gstr.breaks[0] = 0
	br := 0
	for breaker.Next() {
		br += len(breaker.Bytes())
		TC().Infof("next grapheme = '%s'", breaker.Text())
		gstr.breaks = append(gstr.breaks, uint8(br))
	}
	TC().Errorf("breaker error = %v", breaker.Err())
	return gstr
}

func (gstr *shortString) Nth(n int) string {
	if n < 0 || n > max(len(gstr.breaks)-2, 0) {
		panic(fmt.Sprintf("grapheme string index out of bounds, [%d] in [0:%d]",
			n, max(len(gstr.breaks)-2, 0)))
	} else if len(gstr.breaks) < 2 {
		return ""
	}
	l, r := gstr.breaks[n], gstr.breaks[n+1]
	return gstr.content[l:r]
}

func (gstr *shortString) Len() int {
	if len(gstr.breaks) < 2 {
		return 0
	}
	return len(gstr.breaks) - 1
}

// --- Mid version -----------------------------------------------------------

type midString struct {
	content string
	breaks  []uint16
}

func makeMidString(s string) String {
	gstr := &midString{content: s}
	breaker := prepareBreaking(s)
	if breaker == nil {
		return gstr
	}
	gstr.breaks = make([]uint16, 1, len(s)/4)
	gstr.breaks[0] = 0
	br := 0
	for breaker.Next() {
		br += len(breaker.Bytes())
		TC().Infof("next grapheme = '%s'", breaker.Text())
		gstr.breaks = append(gstr.breaks, uint16(br))
	}
	TC().Errorf("breaker error = %v", breaker.Err())
	return gstr
}

func (gstr *midString) Nth(n int) string {
	if n < 0 || n > max(len(gstr.breaks)-2, 0) {
		panic(fmt.Sprintf("grapheme string index out of bounds, [%d] in [0:%d]",
			n, max(len(gstr.breaks)-2, 0)))
	} else if len(gstr.breaks) < 2 {
		return ""
	}
	l, r := gstr.breaks[n], gstr.breaks[n+1]
	return gstr.content[l:r]
}

func (gstr *midString) Len() int {
	if len(gstr.breaks) < 2 {
		return 0
	}
	return len(gstr.breaks) - 1
}

// ---------------------------------------------------------------------------

func prepareBreaking(s string) *segment.Segmenter {
	breaker := makeGraphemeBreaker()
	start, _ := positionOfFirstLegalRune(s)
	if start < 0 {
		TC().Errorf("cannot create grapheme string from invalid rune input")
	}
	breaker.Init(&rr{input: s[start:], pos: 0})
	return breaker
}

// return legal start position and cut-off prefix, if any
func positionOfFirstLegalRune(s string) (int, []byte) {
	i, l, start := 0, len(s), -1
	for i < l {
		if utf8.RuneStart(s[i]) {
			r, _ := utf8.DecodeRuneInString(s[i:])
			if r != utf8.RuneError {
				start = i
			}
			break
		}
	}
	TC().Debugf("start index = %d", start)
	return start, []byte(s[:i])
}

func makeGraphemeBreaker() *segment.Segmenter {
	onGraphemes := NewBreaker()
	segm := segment.NewSegmenter(onGraphemes)
	return segm
}

type rr struct {
	input string
	pos   int
}

func (reader *rr) ReadRune() (r rune, size int, err error) {
	r, size = utf8.DecodeRuneInString(reader.input)
	TC().Debugf("read rune %v with size %d", r, size)
	if r == utf8.RuneError {
		err = io.EOF
		return
	}
	reader.input = reader.input[size:]
	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
