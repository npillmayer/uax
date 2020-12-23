package bidi

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/npillmayer/gorgo/lr/sppf"
	"github.com/npillmayer/gorgo/terex"

	"github.com/npillmayer/gorgo/lr/scanner"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/uax/ucd"
	"golang.org/x/text/unicode/bidi"
)

func TestScanner(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	input := "Hell\u0302o (吾輩は World!)"
	//input := " Hello 123.0 \u0633\u0644\u0627\u0645 89" // Arabic
	//input := "sum = $12453.00"
	reader := strings.NewReader(input)
	sc := NewScanner(reader)
	cnt := 0
	for {
		cnt++
		tokval, token, pos, _ := sc.NextToken(scanner.AnyToken)
		t.Logf("token '%s' at %d = %s", token, pos, ClassString(bidi.Class(tokval)))
		sc.bd16stack.dump()
		if tokval == scanner.EOF {
			break
		}
	}
	if cnt != 9 {
		t.Errorf("Expected scanner to return 9 tokens, was %d", cnt)
	}
}

var inputs = []string{
	//"car wash!",
	//"\u0633\u0644\u0627\u0645 89", // Arabic
	//"car THECAR arabic!",
	//"ab!",
	//"car THE CAR in ARABIC SCRIPT!",
	//"aber (ab!)",
	//"12.453,45€",
	//"sum = 12453€",
	//"sum = $12453.00",
	//"hello w\u0302orld !",
	//"(sum = $12453.00) OK?",
	//`he said "THE VALUES ARE 123, 456, 789, OK".`,
	`VALUES 123, 456, OK.`,
}

func TestSelected(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	T().SetTraceLevel(tracing.LevelDebug)
	gtrace.SyntaxTracer = gologadapter.New()
	//gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelDebug)
	gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelDebug)
	for i, input := range inputs {
		scan := NewScanner(strings.NewReader(input), Testing(true))
		accept, tree, err := Parse(scan)
		if err != nil {
			t.Error(err)
		}
		if !accept {
			t.Fatalf("Test input #%d: not recognized as a valid Bidi run", i)
		} else {
			dotty(tree, t)
		}
	}
}

func TestTermR(t *testing.T) {
	//input := "hello world"
	input := "check (sum = $12453.00)?"
	gtrace.CoreTracer = gologadapter.New()
	T().SetTraceLevel(tracing.LevelDebug)
	gtrace.SyntaxTracer = gologadapter.New()
	gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelInfo)
	scan := NewScanner(strings.NewReader(input), Testing(true))
	accept, tree, err := Parse(scan)
	if err != nil {
		t.Error(err)
	}
	if !accept {
		t.Fatalf("Test input '%s': not recognized as a valid Bidi run", input)
	}
	T().Infof("OK, tree type is %T", tree)
	ast, _, err := AST(tree, earleyTokenRetriever(getParser()))
	if err != nil {
		t.Error(err)
	}
	terex.Elem(ast).Dump(tracing.LevelInfo)
	if ast == nil {
		t.Errorf("AST is empty")
	}
}

func TestRewriting(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	T().SetTraceLevel(tracing.LevelError)
	gtrace.SyntaxTracer = gologadapter.New()
	gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelInfo)
	terex.InitGlobalEnvironment()
	// _, env := terexlang.Parse("(1 a)")
	// if env != nil {
	// 	t.Logf("=== Environment ===")
	// 	t.Logf(env.Dump())
	// }
	// t.Fail()
}

func TestCharacterTestfile(t *testing.T) {
	// teardown := gotestingadapter.RedirectTracing(t)
	// defer teardown()
	gtrace.CoreTracer = gologadapter.New()
	T().SetTraceLevel(tracing.LevelDebug)
	tf := ucd.OpenTestFile("./BidiCharacterTest.txt", t)
	defer tf.Close()
	failcnt, i, from, to := 0, 1, 1, 2
	for tf.Scan() {
		if i >= from {
			fields := strings.Split(tf.Text(), ";")
			if len(fields) >= 2 {
				s := readHex(fields[0])
				if len(s) > 0 {
					i++
					levels := readLevels(fields[3])
					T().Debugf("[%3d] Input = \"%v\", levels=%v | %d", i, s, levels, len(levels))
					if !executeSingleTest(t, s, i) {
						failcnt++
					}
				}
			}
		}
		if i >= to {
			break
		}
	}
	if err := tf.Err(); err != nil {
		t.Errorf("reading input: %s", err)
	}
	t.Logf("%d TEST CASES OUT of %d FAILED", failcnt, i-from+1)
}

func readHex(inp string) string {
	sc := bufio.NewScanner(strings.NewReader(inp))
	sc.Split(bufio.ScanWords)
	run := bytes.NewBuffer(make([]byte, 0, 20))
	for sc.Scan() {
		token := sc.Text()
		n, _ := strconv.ParseUint(token, 16, 64)
		run.WriteRune(rune(n))
	}
	//fmt.Printf("input = '%s'\n", inp.String())
	//fmt.Printf("output = %#v\n", out)
	return run.String()
}

func readLevels(inp string) []int {
	sc := bufio.NewScanner(strings.NewReader(inp))
	sc.Split(bufio.ScanWords)
	l := make([]int, 0, len(inp)/2+1)
	for sc.Scan() {
		i := sc.Text()
		n, _ := strconv.ParseInt(i, 10, 64)
		l = append(l, int(n))
	}
	return l
}

//func executeSingleTest(t *testing.T, seg *segment.Segmenter, tno int, in string, out []string) bool {
func executeSingleTest(t *testing.T, inp string, i int) bool {
	scan := NewScanner(strings.NewReader(inp), Testing(true))
	accept, _, err := Parse(scan)
	if err != nil {
		t.Error(err)
	}
	if !accept {
		t.Fatalf("Test input #%d: not recognized as a valid Bidi run", i)
	} else {
		//dotty(tree, t)
	}
	return accept
}

// ---------------------------------------------------------------------------

func dotty(parsetree *sppf.Forest, t *testing.T) {
	tmpfile, err := ioutil.TempFile(".", "bidi-*.dot")
	if err != nil {
		t.Errorf("cannot open tmp file")
	} else {
		sppf.ToGraphViz(parsetree, tmpfile)
		t.Logf("Exported parse tree to %s", tmpfile.Name())
	}
}
