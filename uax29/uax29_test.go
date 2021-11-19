package uax29_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/internal/ucdparse"
	"github.com/npillmayer/uax/segment"
	"github.com/npillmayer/uax/uax29"
)

func ExampleWordBreaker() {
	onWords := uax29.NewWordBreaker(1)
	segmenter := segment.NewSegmenter(onWords)
	segmenter.Init(strings.NewReader("Hello World🇩🇪!"))
	for segmenter.Next() {
		fmt.Printf("'%s'\n", segmenter.Text())
	}
	// Output: 'Hello'
	// ' '
	// 'World'
	// '🇩🇪'
	// '!'
}

func TestWordBreaks1(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	// gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	onWords := uax29.NewWordBreaker(1)
	segmenter := segment.NewSegmenter(onWords)
	segmenter.Init(strings.NewReader("Hello World🇩🇪!"))
	n := 0
	for segmenter.Next() {
		t.Logf("'%s'\n", segmenter.Text())
		n++
	}
	if n != 5 {
		t.Errorf("Expected # of segments to be 5, is %d", n)
	}
}

func TestWordBreaks2(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	onWords := uax29.NewWordBreaker(1)
	segmenter := segment.NewSegmenter(onWords)
	segmenter.Init(strings.NewReader("lime-tree"))
	n := 0
	for segmenter.Next() {
		p1, p2 := segmenter.Penalties()
		t.Logf("'%s'  (p=%d|%d)", segmenter.Text(), p1, p2)
		n++
	}
	if n != 3 {
		t.Errorf("Expected # of segments to be 3, is %d", n)
	}
	//t.Fail()
}

func TestWordBreakTestFile(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelError)
	//
	onWordBreak := uax29.NewWordBreaker(1)
	seg := segment.NewSegmenter(onWordBreak)
	//seg.BreakOnZero(true, false)
	tf := ucdparse.OpenTestFile("./WordBreakTest.txt", t)
	defer tf.Close()
	failcnt, i, from, to := 0, 0, 1, 1900
	for tf.Scan() {
		i++
		if i >= from {
			gtrace.CoreTracer.Infof(tf.Comment())
			in, out := ucdparse.BreakTestInput(tf.Text())
			if !executeSingleTest(t, seg, i, in, out) {
				failcnt++
				//t.Fatalf("test #%d failed", i)
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

func executeSingleTest(t *testing.T, seg *segment.Segmenter, tno int, in string, out []string) bool {
	seg.Init(strings.NewReader(in))
	i := 0
	ok := true
	for seg.Next() {
		if len(out) <= i {
			t.Errorf("test #%d: number of segments too large: %d > %d", tno, i+1, len(out))
		} else if out[i] != seg.Text() {
			t.Errorf("test #%d: '%+q' should be '%+q'", tno, seg.Bytes(), out[i])
			ok = false
		}
		i++
	}
	return ok
}
