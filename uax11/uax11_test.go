package uax11

import (
	"testing"
	"unicode/utf8"

	"github.com/npillmayer/uax/grapheme"
	"github.com/npillmayer/uax/internal/tracing"
	"golang.org/x/text/width"
)

/*
func TestTables(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	chars := [...]rune{
		'A',    // LATIN CAPITAL LETTER A           => Na
		0x05BD, // HEBREW POINT METEG               => N
		0x2223, // DIVIDES                          => A
		0x3008, // LEFT ANGLE BRACKET               => W
		0xFF41, // FULLWIDTH LATIN SMALL LETTER A   => F
	}
	cats := [...]Category{Na, N, A, W, F}
	for i, c := range chars {
		cat := WidthCategory(c)
		if cat != cats[i] {
			t.Errorf("expected width category of %#U to be %d, is %d", c, cats[i], cat)
		}
	}
}
*/

func TestWidth(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	chars := [...]rune{
		'A',    // LATIN CAPITAL LETTER A           => Na
		0x05BD, // HEBREW POINT METEG               => N
		0x2223, // DIVIDES                          => A
		0x3008, // LEFT ANGLE BRACKET               => W
		0xFF41, // FULLWIDTH LATIN SMALL LETTER A   => F
	}
	ctx := LatinContext
	buf := make([]byte, 10)
	ww := 0
	for i, r := range chars {
		//cat := WidthCategory(r)
		cat := width.LookupRune(r).Kind()
		len := utf8.EncodeRune(buf, r)
		//t.Logf("cat(%#U) = %d", r, cat)
		w := Width(buf, ctx)
		t.Logf("%d: %#U:'%08x' (%d) => %d", i, r, buf[:len], cat, w)
		ww += w
	}
	if ww != 7 {
		t.Errorf("expected accumulated width of 5 runes to be 7, is %d", ww)
	}
}

func TestContext(t *testing.T) {
	tracing.SetTestingLog(t)

	//context := &Context{Locale: "zh-uig"}
	context := &Context{Locale: "zh-HK"}
	_ = Width([]byte("世"), context)
	t.Logf("%v", context.Script)
	//t.Fail()
}

func TestString(t *testing.T) {
	tracing.SetTestingLog(t)

	input := "A (世). "
	buf := make([]byte, 10)
	len := utf8.EncodeRune(buf, 0x1f600)
	input = input + string(buf[:len])
	t.Logf("input string = '%v'", input)
	ctx := EastAsianContext
	s := grapheme.StringFromString(input)
	w := StringWidth(s, ctx)
	if w != 10 {
		t.Errorf("expected fixed width length of string to be 10, is %d", w)
	}
	//t.Fail()
}

func TestScripts(t *testing.T) {
	tracing.SetTestingLog(t)

	input := []struct {
		S    string
		N, W int
	}{
		{"\u05d1\u05E8\u05E9\u05d0\u05D9\u05EA", 6, 12}, // ברשאית
		{"\u05EA\u05BC", 1, 2},                          // תּ
		{"\u05D0\u200D\u05DC", 2, 4},                    // א‍ל
	}
	for i, inp := range input {
		s := grapheme.StringFromString(inp.S)
		w1 := StringWidth(s, LatinContext)
		w2 := StringWidth(s, EastAsianContext)
		t.Logf("%d (%s) : %d bytes, %d | %d", i, inp.S, len(inp.S), w1, w2)
		if w1 != inp.N || w2 != inp.W {
			t.Errorf("expected widths to be (%d|%d), are (%d|%d)", inp.N, inp.W, w1, w2)
		}
	}
	//t.Fail()
}
