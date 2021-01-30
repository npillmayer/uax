package uax11

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
)

func TestWidth(t *testing.T) {
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
	t.Fail()
}
