package bidi

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
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

func TestScannerMarkup(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := strings.NewReader("the fox")
	markup := func(pos uint64) int {
		if pos == 4 {
			return MarkupPDILRI
		}
		return 0
	}
	t.Logf("markup PDI LRI = %d", MarkupPDILRI)
	t.Logf("markup PDI     = %d", MarkupPDI)
	t.Logf("markup LRI     = %d", MarkupLRI)
	scnr := newScanner(input, markup, TestMode(true))
	pipe := make(chan scrap, 0)
	go scnr.Scan(pipe)
	n := 0
	scraps := "produced scraps:"
	for s := range pipe {
		if s.bidiclz == cNULL {
			scraps += "\n----------------"
		} else {
			scraps += fmt.Sprintf("\n[%2d] -> %s", n, s)
		}
		n++
		if n == 3 && s.bidiclz != bidi.PDI {
			t.Errorf("expected scrap #3 to be PDI, is %v", s)
		}
		if n == 4 && s.bidiclz != bidi.LRI {
			t.Errorf("expected scrap #4 to be LRI, is %v", s)
		}
	}
	t.Logf(scraps)
}

func TestScannerScraps(t *testing.T) {
	// gtrace.CoreTracer = gologadapter.New()
	teardown := testconfig.QuickConfig(t)
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
		scnr := newScanner(input, nil, TestMode(true))
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
	teardown := testconfig.QuickConfig(t)
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
	teardown := testconfig.QuickConfig(t)
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
	teardown := testconfig.QuickConfig(t)
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
		{l: 40, r: 42, bidiclz: bidi.R},
		{l: 42, r: 45, bidiclz: bidi.EN},
		{l: 45, r: 45, bidiclz: bidi.PDI},
	}
	scraps[2].children = append(scraps[2].children, chRL[:])
	t.Logf("scraps=%v,   emb=%v", scraps, LeftToRight)
	rev := reorder(scraps[:], 0, len(scraps), LeftToRight)
	t.Logf("   rev=%v", rev)
	if rev[0].bidiclz != bidi.R || rev[0].l != 15 {
		t.Errorf("expected reordered run[0] to be [15-R-20], is %v", rev[0])
	}
	if len(rev[0].children) == 0 || rev[0].children[0][1].bidiclz != bidi.EN {
		t.Errorf("expected child run[1] to be EN, is %v", rev[0].children[0][1])
	}
}

func TestRunConcat(t *testing.T) {
	run1 := Run{Dir: LeftToRight, Length: 10}
	run1.scraps = []scrap{{l: 0, r: 10, bidiclz: bidi.L}}
	run2 := Run{Dir: LeftToRight, Length: 15}
	run2.scraps = []scrap{{l: 10, r: 20, bidiclz: bidi.EN},
		{l: 20, r: 25, bidiclz: bidi.L}}
	run1.concat(run2)
	t.Logf("run1=%v", run1)
	if len(run1.scraps) != 2 {
		t.Errorf("expected run1 to own 2 scraps, doesn't: %v", run1.scraps)
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
		{l: 35, r: 36, bidiclz: bidi.RLI},
		{l: 36, r: 40, bidiclz: bidi.R},
		{l: 40, r: 44, bidiclz: bidi.EN},
		{l: 44, r: 45, bidiclz: bidi.PDI},
	}
	scraps[3].children = append(scraps[2].children, chRL[:])
	t.Logf("scraps=%v", scraps)
	rev := reorder(scraps[:], 0, len(scraps), LeftToRight)
	t.Logf("   rev=%v", rev)
	T().Debugf("=====================================")
	flat := flatten(rev, LeftToRight)
	t.Logf("flat runs = %v", flat)
	// [(R2L 5 15…20|R) (L2R 5 10…15|EN) (R2L 10 0…10|R) (L2R 15 20…35|L)
	//  (R2L 1 35…36|RLI) (L2R 4 40…44|EN) (R2L 5 36…40|R 44…45|PDI) (L2R 5 45…50|L)]
	if len(flat) != 8 {
		t.Errorf("expected 8 runs, have %d", len(flat))
	}
	if flat[4].Length != 1 || flat[4].Dir != RightToLeft {
		t.Errorf("expected run 4 to be R2L, is %v", flat[4])
	}
}

func TestSplitSingle(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var scraps = [...]scrap{
		{l: 0, r: 0, bidiclz: bidi.LRI},
		{l: 0, r: 35, bidiclz: bidi.L},
		{l: 35, r: 50, bidiclz: bidi.R},
		{l: 50, r: 51, bidiclz: bidi.L},
		{l: 51, r: 51, bidiclz: bidi.PDI},
	}
	prefix, suffix := split(scraps[:], 38)
	if len(prefix) != 3 || len(suffix) != 4 {
		t.Errorf("expected split into 3|4, is %d|%d", len(prefix), len(suffix))
	}
}

func TestSplit(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var scraps = [...]scrap{
		{l: 0, r: 0, bidiclz: bidi.LRI},
		{l: 0, r: 10, bidiclz: bidi.R},
		{l: 10, r: 15, bidiclz: bidi.EN},
		{l: 15, r: 20, bidiclz: bidi.R},
		{l: 20, r: 50, bidiclz: bidi.L},
		{l: 50, r: 50, bidiclz: bidi.PDI},
	}
	var chRL = [...]scrap{
		{l: 35, r: 35, bidiclz: bidi.RLI},
		{l: 35, r: 40, bidiclz: bidi.R},
		{l: 40, r: 45, bidiclz: bidi.EN},
		{l: 45, r: 45, bidiclz: bidi.PDI},
	}
	scraps[4].children = append(scraps[2].children, chRL[:])
	t.Logf("scraps=%v", scraps)
	prefix, suffix := split(scraps[:], 37)
	if len(prefix) != 5 || len(suffix) != 3 {
		t.Errorf("expected split into 5|3, is %d|%d", len(prefix), len(suffix))
	}
}

func TestScannerBrackets(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	input := strings.NewReader("hi (YOU[])")
	scnr := newScanner(input, nil, TestMode(true))
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
	levels := ResolveParagraph(reader, nil, TestMode(true))
	fmt.Printf("resulting levels = %s\n", levels)
	if len(levels.scraps) != 3 || levels.scraps[1].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L, is '%s'", levels.scraps)
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
	levels := ResolveParagraph(reader, nil, TestMode(true))
	fmt.Printf("resulting levels = %s\n", levels)
	if len(levels.scraps) != 5 || levels.scraps[2].bidiclz != bidi.R {
		t.Errorf("expected ordering to be L + R + L, is '%s'", levels.scraps)
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
	levels := ResolveParagraph(reader, nil, TestMode(true))
	fmt.Printf("resulting levels = %s\n", levels)
	if len(levels.scraps) != 3 || levels.scraps[1].bidiclz != bidi.L {
		t.Errorf("expected ordering to be L, is '%v'", levels.scraps)
	}
	if len(levels.scraps[1].children) != 1 {
		t.Errorf("expected sub-IRS to be wrapped as a child, isn't")
	}
	fmt.Printf("  %s → %v\n", levels.scraps[1], levels.scraps[1].children[0])
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
	levels := ResolveParagraph(reader, nil, TestMode(true))
	fmt.Printf("resulting levels = %s\n", levels)
	if len(levels.scraps) != 4 || levels.scraps[1].bidiclz != bidi.L {
		t.Errorf("expected levels to be L + R, is '%v'", levels.scraps)
	}
}

// ===========================================================================
// Examples from the UAX#9 paper, section 3.4 "Reordering Resolved Levels", L2.
// ===========================================================================

// First try to resolve level runs for all the examples.

func TestResolveCar1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := "car means CAR."
	t.Logf("input of length %d is '%s'", len(input), input)
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true))
	t.Logf("resulting levels     = %v\n", levels)
	if len(levels.scraps) != 5 {
		t.Fatalf("expected 5 level runs, have %d", len(levels.scraps))
	}
	if levels.scraps[1].len() != 10 {
		t.Errorf("expected L run to be of length 10, is %v", levels.scraps[1])
	}
}

func TestResolveCar2(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	input := "<car MEANS CAR.="
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true))
	// [0-LRI-0] [0-L-16] [16-PDI-16]
	// sub: [[[0.RLI] [1-L-4] [4-R-15] [15.PDI]]]
	t.Logf("resulting levels     = %v\n", levels)
	if len(levels.scraps) != 3 {
		t.Fatalf("expected 3 level runs, have %d", len(levels.scraps))
	}
	t.Logf("resulting sub-levels = %v\n", levels.scraps[1].children)
	if len(levels.scraps[1].children) != 1 {
		t.Fatalf("expected 1 sub level run, have %d", len(levels.scraps[1].children))
	}
	if levels.scraps[1].children[0][1].len() != 3 {
		t.Errorf("expected L-level for 'car' to be of length 3, have car=%v",
			levels.scraps[1].children[0][1])
	}
}

func TestResolveCar3(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := "he said “<car MEANS CAR=.” “<IT DOES=,” she agreed."
	t.Logf("input with len=%d : '%v'", len(input), input)
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true))
	t.Logf("resulting levels     = %v\n", levels)
	if len(levels.scraps) != 3 {
		t.Fatalf("expected to get 3 level runs, got %d", len(levels.scraps))
	}
	t.Logf("      sub levels     = %v\n", levels.scraps[1].children)
	// [0-LRI-0] [0-L-59] [59-PDI-59]
	// [[[11.RLI] [12-L-15] [15-R-25] [25.PDI]] [[34.RLI] [35-R-42] [42.PDI]]]
	if len(levels.scraps[1].children) != 2 {
		t.Errorf("expected top level to have 2 child runs, has %d", len(levels.scraps[1].children))
	}
	if levels.scraps[1].children[0][1].bidiclz != bidi.L {
		t.Errorf("expected 1st sub-level to have L-run at pos.2, hasn't: %s",
			levels.scraps[1].children[0][1])
	}
}

func TestResolveCar4(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := "DID YOU SAY ’>he said “<car MEANS CAR=”=‘?"
	t.Logf("input with len=%d : '%v'", len(input), input)
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true), DefaultDirection(RightToLeft))
	t.Logf("resulting levels     = %v\n", levels)
	if len(levels.scraps) != 3 {
		t.Fatalf("expected to get 3 level runs, got %d", len(levels.scraps))
	}
	t.Logf("      sub levels     = %v\n", levels.scraps[1].children)
	//     top: [0-RLI-0] [0-R-50] [50-PDI-50]
	//     sub: [[[15.LRI] [16-L-45] [45.PDI]]]
	// sub-sub: [[[27.RLI] [28-L-31] [31-R-41] [41.PDI]]]
	if len(levels.scraps[1].children) != 1 {
		t.Errorf("expected top level to have 1 child run, has %d", len(levels.scraps[1].children))
	}
	t.Logf("  sub sub levels     = %v\n", levels.scraps[1].children[0][1].children)
	if len(levels.scraps[1].children[0][1].children[0]) != 4 {
		t.Fatalf("expected 2nd sub-level to have 4 level runs, got %d",
			len(levels.scraps[1].children[0][1].children[0]))
	}
	if levels.scraps[1].children[0][1].children[0][1].bidiclz != bidi.L {
		t.Errorf("expected 2nd sub-level to have L-run at pos.2, hasn't: %s",
			levels.scraps[1].children[0][1].children[0][1])
	}
}

func TestOrderCar1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := "car means CAR."
	t.Logf("input of length %d is '%s'", len(input), input)
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true))
	fmt.Printf("resulting levels     = %v\n", levels)
	runs := levels.Reorder()
	fmt.Printf("resulting runs       = %v\n", runs.Runs)
	if len(runs.Runs) != 3 {
		t.Errorf("expected 3 resulting runs, but have %d", len(runs.Runs))
	}
	// [(L2R 10 0…10|L) (R2L 3 10…13|R) (L2R 1 13…14|L)]
	if runs.Runs[0].Length != 10 {
		t.Errorf("expected L2R-run to end at 10, doesn't: %v", runs.Runs[0])
	}
	out := applyOrder([]byte(input), runs)
	disp := display(out)
	fmt.Printf("display              = \"%s\"\n", disp)
	target := "car means RAC."
	if disp != target {
		t.Errorf("expected display output \"%s\", is \"%s\"", target, disp)
	}
}

func TestOrderCar2(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := "<car MEANS CAR.="
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true))
	runs := levels.Reorder()
	fmt.Printf("resulting runs       = %v\n", runs.Runs)
	if len(runs.Runs) != 3 {
		t.Errorf("expected 4 resulting runs, but have %d", len(runs.Runs))
	}
	// [(R2L 12 0…1|RLI 4…15|R) (L2R 3 1…4|L) (R2L 1 15…16|PDI)]
	if runs.Runs[1].Length != 3 {
		t.Errorf("expected L2R-run of length 3, is: %v", runs.Runs[1])
	}
	out := applyOrder([]byte(input), runs)
	disp := display(out)
	fmt.Printf("display              = \"%s\"\n", disp)
	target := ".RAC SNAEM car"
	if disp != target {
		t.Errorf("expected display output \"%s\", is \"%s\"", target, disp)
	}
}

func TestOrderCar3(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	input := "he said “<car MEANS CAR=.” “<IT DOES=,” she agreed."
	t.Logf("input = '%s'", input)
	reader := strings.NewReader(input)
	levels := ResolveParagraph(reader, nil, TestMode(true))
	//
	runs := levels.Reorder()
	t.Logf("resulting runs       = %v\n", runs.Runs)
	if len(runs.Runs) != 7 {
		t.Errorf("expected 7 resulting bidi runs, but have %d", len(runs.Runs))
	}
	if runs.Runs[1].Length != 11 {
		t.Errorf("expected R2L-run of length 11, is: %v", runs.Runs[1])
	}
	// [(L2R 11 0…11|L) (R2L 11 11…12|RLI 15…25|R) (L2R 3 12…15|L) (R2L 1 25…26|PDI)
	//  (L2R 8 26…34|L) (R2L 9 34…43|RLI) (L2R 16 43…59|L)]
	out := applyOrder([]byte(input), runs)
	disp := display(out)
	t.Logf("display              = \"%s\"\n", disp)
	target := "he said “RAC SNAEM car.” “SEOD TI,” she agreed."
	if disp != target {
		t.Logf("expected display output \"%s\"", target)
		t.Errorf("is \"%s\"", disp)
	}
}

func NoTestUAXFile(t *testing.T) {
	// gtrace.CoreTracer = gotestingadapter.New()
	// teardown := gotestingadapter.RedirectTracing(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// reader := strings.NewReader("hel<lo WORLD")
	// ordering := ResolveParagraph(reader, nil, TestMode(true))
	// fmt.Printf("resulting levels = %s\n", ordering)
	// if len(ordering.scraps) != 4 || ordering.scraps[1].bidiclz != bidi.L {
	// 	t.Errorf("expected ordering to be L + R, is '%v'", ordering.scraps)
	// }
	//
	readBidiTests("./uaxbiditest/BidiCharacterTest.txt")
}

func TestTest(t *testing.T) {
	input := "he said “<car MEANS CAR=.” “<IT DOES=,” she agreed."
	s := []byte(input[:11])
	t.Logf("s = '%s'", s)
	t.Logf("reversing '%s' = '%s'", s, reverseString(s))
	//t.Fail()
}

// ---------------------------------------------------------------------------

func applyOrder(text []byte, order *Ordering) []byte {
	var r []byte = make([]byte, 0, len(text))
	for _, run := range order.Runs {
		for _, s := range run.scraps {
			b := text[s.l:s.r]
			if run.Dir == RightToLeft {
				b = reverseString(b)
			}
			r = append(r, b...)
		}
	}
	return r
}

func reverseString(b []byte) []byte {
	str := string(b)
	s := []rune(str)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return []byte(string(s))
}

func display(b []byte) string {
	str := []rune(string(b))
	s := ""
	for _, c := range str {
		if c == '<' || c == '>' || c == '=' {
			continue
		}
		s += string(c)
	}
	return s
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
