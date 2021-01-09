package bidi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/text/unicode/bidi"

	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestClasses(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	t.Logf("L = %s", classString(bidi.L))
	if classString(bidi.L) != "L" {
		t.Errorf("string for L not as expected")
	}
	t.Logf("NI = %s", classString(cNI))
	if classString(cNI) != "NI" {
		t.Errorf("string for NI not as expected")
	}
	t.Logf("BRACKC = %s", classString(cBRACKC))
	if classString(cBRACKC) != "BRACKC" {
		t.Errorf("string for BRACKC not as expected")
	}
	t.Logf("MAX = %s", classString(cMAX))
	if classString(cMAX) != "<max>" {
		t.Errorf("string for MAX not as expected")
	}
}

func TestScannerScraps(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelError)
	//
	inputs := []struct {
		str string
		cnt int
	}{
		{str: "hello 12.345", cnt: 5},
		{str: "Hello (123)", cnt: 6},
		{str: "smith (fabrikam ARABIC) HEBREW", cnt: 9},
		{str: "AB(CD[&ef])gh", cnt: 9},
	}
	for i, inp := range inputs {
		input := strings.NewReader(inp.str)
		scnr := newScanner(input, TestMode(true))
		pipe := make(chan scrap, 0)
		go scnr.Scan(pipe)
		n := 0
		scraps := "produced scraps:"
		for s := range pipe {
			if s.bidiclz == cNULL {
				scraps += "\n----------------"
			} else {
				scraps += fmt.Sprintf("\n[%d %2d] -> %s", i, n, s)
			}
			n++
		}
		if n-1 != inp.cnt {
			t.Logf("scanner test for [%d] \"%s\"", i, inp.str)
			t.Logf(scraps)
			t.Errorf("ERROR: expected scanner to produce %d scraps, have %d", inp.cnt, n-1)
		}
	}
}

func TestScannerBrackets(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	input := strings.NewReader("hi (YOU[])")
	scnr := newScanner(input, TestMode(true))
	pipe := make(chan scrap, 0)
	go scnr.Scan(pipe)
	for s := range pipe {
		t.Logf("-> %s", s)
		if s.bidiclz == cBRACKC {
			pair, found := scnr.bd16.FindBracketPairing(s) //, Closing)
			if !found {
				t.Errorf("expected closing bracket %s to form a pairing, did not", s)
			}
			t.Logf("pairing found: %v", pair)
		}
	}
}

func TestSimple(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	//
	reader := strings.NewReader("hello 123.45")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting ordering = %s\n", ordering)
	if len(ordering.scraps) != 1 || ordering.scraps[0].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L + NI + L, is '%s'", ordering)
	}
}

func TestBrackets(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("hello (World)")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting ordering = %s\n", ordering)
	if len(ordering.scraps) != 3 || ordering.scraps[2].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L + R + L, is '%s'", ordering)
	}
}

func TestTODO(t *testing.T) {
	//gtrace.CoreTracer = gotestingadapter.New()
	gtrace.CoreTracer = gologadapter.New()
	//teardown := gotestingadapter.RedirectTracing(t)
	//defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("hello")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting ordering = %s\n", ordering)
	// if len(ordering.scraps) != 3 || ordering.scraps[2].bidiclz != bidi.L {
	// 	t.Errorf("expected ordering to be L + R + L, is '%s'", ordering)
	// }
}
