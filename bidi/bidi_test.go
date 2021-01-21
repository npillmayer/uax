package bidi

import (
	"bufio"
	"fmt"
	"log"
	"os"
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

func TestSimpleReverse(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var scraps = [...]scrap{
		{l: 0, r: 5, bidiclz: bidi.L},
		{l: 5, r: 10, bidiclz: bidi.R},
		{l: 10, r: 15, bidiclz: bidi.EN},
		{l: 15, r: 20, bidiclz: bidi.R},
	}
	t.Logf("scraps=%v", scraps)
	rev := reverse(scraps[:], 0, len(scraps))
	t.Logf("   rev=%v", rev)
	if rev[0].bidiclz != bidi.R {
		t.Fatalf("expected reversed run to have R at position 0, has L")
	}
}

func TestSimpleL2RReorder(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var scraps = [...]scrap{
		{l: 0, r: 5, bidiclz: bidi.L},
		{l: 5, r: 10, bidiclz: bidi.R},
		{l: 10, r: 15, bidiclz: bidi.EN},
		{l: 15, r: 20, bidiclz: bidi.R},
	}
	t.Logf("scraps=%v", scraps)
	rev := reorder(scraps[:], 0, len(scraps), LeftToRight)
	t.Logf("   rev=%v", rev)
	if rev[0].bidiclz != bidi.L {
		t.Fatalf("expected reversed run to have L at position 0, has R")
	}
	if rev[1].r != 20 {
		t.Fatalf("expected 2nd (R) to end at position 20, is %d", rev[1].r)
	}
}

func TestRecursiveL2RReorder(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var scraps = [...]scrap{
		{l: 0, r: 10, bidiclz: bidi.R},
		{l: 10, r: 15, bidiclz: bidi.EN},
		{l: 15, r: 20, bidiclz: bidi.R},
		{l: 20, r: 50, bidiclz: bidi.L},
	}
	var chRL = [...]scrap{
		{l: 40, r: 40, bidiclz: bidi.RLI},
		{l: 40, r: 40, bidiclz: bidi.R},
		{l: 40, r: 45, bidiclz: bidi.EN},
		{l: 45, r: 45, bidiclz: bidi.PDI},
	}
	scraps[2].children = append(scraps[2].children, chRL[:])
	t.Logf("scraps=%v", scraps)
	rev := reorder(scraps[:], 0, len(scraps), LeftToRight)
	t.Logf("   rev=%v", rev)
	if rev[0].bidiclz != bidi.R {
		t.Fatalf("expected reordered run to have R at position 0, has L")
	}
}

func TestFlatten1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var scraps = [...]scrap{
		{l: 0, r: 10, bidiclz: bidi.R},
		{l: 10, r: 15, bidiclz: bidi.EN},
		{l: 15, r: 20, bidiclz: bidi.R},
		{l: 20, r: 50, bidiclz: bidi.L},
	}
	var chRL = [...]scrap{
		{l: 35, r: 35, bidiclz: bidi.RLI},
		{l: 35, r: 40, bidiclz: bidi.R},
		{l: 40, r: 45, bidiclz: bidi.EN},
		{l: 45, r: 45, bidiclz: bidi.PDI},
	}
	scraps[3].children = append(scraps[2].children, chRL[:])
	t.Logf("scraps=%v", scraps)
	rev := reorder(scraps[:], 0, len(scraps), LeftToRight)
	t.Logf("   rev=%v", rev)
	T().Debugf("=====================================")
	flat := flatten(rev, LeftToRight)
	t.Logf("flat runs = %v", flat)
	//  [{R2L 15 20} {L2R 10 15} {R2L 0 10} {L2R 20 35} {R2L 35 40} {L2R 40 50}]
	if flat[len(flat)-1].L != 40 {
		t.Errorf("expected last run to be {L2R 40 50}, is %v", flat[len(flat)-1])
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
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("hello 123.45")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting levels = %s\n", ordering)
	if len(ordering.scraps) != 3 || ordering.scraps[1].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L, is '%s'", ordering.scraps)
	}
}

func TestBrackets(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("hello (WORLD)")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting levels = %s\n", ordering)
	if len(ordering.scraps) != 5 || ordering.scraps[2].bidiclz != bidi.R {
		t.Errorf("expected ordering to be L + R + L, is '%s'", ordering.scraps)
	}
}

func TestIRS(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("hel<lo WORLD=")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting levels = %s\n", ordering)
	if len(ordering.scraps) != 3 || ordering.scraps[1].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L, is '%v'", ordering.scraps)
	}
	if len(ordering.scraps[1].children) != 1 {
		t.Errorf("expected sub-IRS to be wrapped as a child, isn't")
	}
	fmt.Printf("  %s → %v\n", ordering.scraps[1], ordering.scraps[1].children[0])
}

func TestIRSLoner(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("hel<lo WORLD")
	ordering := ResolveParagraph(reader, TestMode(true))
	fmt.Printf("resulting levels = %s\n", ordering)
	if len(ordering.scraps) != 4 || ordering.scraps[1].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L + R, is '%v'", ordering.scraps)
	}
}

func TestUAXFile(t *testing.T) {
	// gtrace.CoreTracer = gotestingadapter.New()
	// teardown := gotestingadapter.RedirectTracing(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// reader := strings.NewReader("hel<lo WORLD")
	// ordering := ResolveParagraph(reader, TestMode(true))
	// fmt.Printf("resulting levels = %s\n", ordering)
	// if len(ordering.scraps) != 4 || ordering.scraps[1].bidiclz != bidi.L {
	// 	t.Errorf("expected ordering to be L + R, is '%v'", ordering.scraps)
	// }
	//
	readBidiTests("./uaxbiditest/BidiCharacterTest.txt")
}

// ---------------------------------------------------------------------------

const batchsize = 1

func readBidiTests(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	header := true
	cnt := batchsize
	for cnt > 0 && scanner.Scan() {
		//fmt.Println(scanner.Text())
		line := scanner.Text()
		if len(line) == 0 {
			header = false
		} else if !header {
			if strings.HasPrefix(line, "#") && !strings.HasSuffix(line, "#") {
				cnt--
				fmt.Println(line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
