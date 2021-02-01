package uax11

import (
	"testing"
	"unicode/utf8"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/emoji"
	"github.com/npillmayer/uax/grapheme"
)

func TestTables(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
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

func TestEnvLocale(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	ctx := ContextFromEnvironment()
	if ctx == nil {
		t.Fatalf("context from environment is nil, should not")
	}
	t.Logf("user environment has locale '%s'", ctx.Locale)
	//t.Fail()
}

func TestWidth(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	emoji.SetupEmojisClasses()
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
		cat := WidthCategory(r)
		len := utf8.EncodeRune(buf, r)
		//t.Logf("cat(%#U) = %d", r, cat)
		w := Width(buf, ctx)
		t.Logf("%d: %#U:'%08x' (%d) => %d", i, r, buf[:len], cat, w)
		ww += w
	}
	if ww != 7 {
		t.Errorf("expected accumalted width of 5 runes to be 7, is %d", ww)
	}
}

func TestContext(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	grapheme.SetupGraphemeClasses()
	//context := &Context{Locale: "zh-uig"}
	context := &Context{Locale: "zh-HK"}
	_ = Width([]byte("世"), context)
	t.Logf("%v", context.Script)
	//t.Fail()
}

func TestString(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelError)
	//
	grapheme.SetupGraphemeClasses()
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
	t.Fail()
}
