package uax14_test

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/segment"
	"github.com/npillmayer/uax/uax14"
	"github.com/npillmayer/uax/ucd"
)

func TestSimpleLineWrap(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	linewrap := uax14.NewLineWrap()
	seg := segment.NewSegmenter(linewrap)
	input := strings.NewReader("Hello World!")
	seg.Init(input)
	cnt := 0
	for seg.Next() {
		cnt++
		t.Logf("segment #%d: %v", cnt, seg.Text())
	}
	if cnt != 2 {
		t.Errorf("expected 2 segments, got %d", cnt)
	}
}

func TestWordBreakTestFile(t *testing.T) {
	//gtrace.CoreTracer = gologadapter.New()
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	linewrap := uax14.NewLineWrap()
	seg := segment.NewSegmenter(linewrap)
	tf := ucd.OpenTestFile("./LineBreakTest.txt", t)
	defer tf.Close()
	//failcnt, i, from, to := 0, 0, 6263, 7000
	failcnt, i, from, to := 0, 0, 0, 7000
	for tf.Scan() {
		i++
		if i >= from {
			//t.Logf(tf.Comment())
			gtrace.CoreTracer.Infof(tf.Comment())
			in, out := ucd.BreakTestInput(tf.Text())
			if !executeSingleTest(t, seg, i, in, out) {
				failcnt++
				t.Errorf("test #%d failed", i)
				//break
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
			ok = false
		} else if out[i] != seg.Text() {
			t.Errorf("test #%d: '%+q' should be '%+q'", tno, seg.Bytes(), out[i])
			ok = false
		}
		i++
	}
	return ok
}
