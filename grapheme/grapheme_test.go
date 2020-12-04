package grapheme

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/uax/segment"
)

//var TC tracing.Trace = gologadapter.New()

func TestGraphemeClasses(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	c1 := LClass
	if c1.String() != "LClass" {
		t.Errorf("String(LClass) should be 'LClass', is %s", c1)
	}
	SetupGraphemeClasses()
	if !unicode.Is(Control, '\t') {
		t.Error("<TAB> should be identified as control character")
	}
	hangsyl := '\uac1c'
	if c := ClassForRune(hangsyl); c != LVClass {
		t.Errorf("Hang syllable GAE should be of class LV, is %s", c)
	}
	if c := ClassForRune(0); c != eot {
		t.Errorf("\\0x00 should be of class eot, is %s", c)
	}
}

func TestGraphemes1(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	onGraphemes := NewBreaker()
	input := bytes.NewReader([]byte("Hello\tWorld"))
	seg := segment.NewSegmenter(onGraphemes)
	seg.Init(input)
	seg.Next()
	t.Logf("Next() = %s\n", seg.Text())
	if seg.Err() != nil {
		t.Errorf("segmenter.Next() failed with error: %s", seg.Err())
	}
}

func TestGraphemes2(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	onGraphemes := NewBreaker()
	input := bytes.NewReader([]byte("Hello\tWorld"))
	seg := segment.NewSegmenter(onGraphemes)
	seg.Init(input)
	for seg.Next() {
		t.Logf("Next() = %s\n", seg.Text())
	}
	if seg.Err() != nil {
		t.Errorf("segmenter.Next() failed with error: %s", seg.Err())
	}
}

func TestGraphemesTestFile(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	TC().SetTraceLevel(tracing.LevelError)
	SetupGraphemeClasses()
	onGraphemes := NewBreaker()
	seg := segment.NewSegmenter(onGraphemes)
	gopath := os.Getenv("GOPATH")
	f, err := os.Open(gopath + "/etc/GraphemeBreakTest.txt")
	if err != nil {
		t.Errorf("ERROR loading " + gopath + "/etc/GraphemeBreakTest.txt\n")
	}
	defer f.Close()
	failcnt, i, from, to := 0, 0, 1, 1000
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		line = strings.TrimSpace(line)
		if line[0] == '#' { // ignore comment lines
			continue
		}
		i++
		if i >= from {
			parts := strings.Split(line, "#")
			testInput, comment := parts[0], parts[1]
			TC().Infof(comment)
			in, out := breakTestInput(testInput)
			if !executeSingleTest(t, seg, i, in, out) {
				failcnt++
			}
		}
		if i >= to {
			break
		}
	}
	if err := scan.Err(); err != nil {
		TC().Errorf("reading input:", err)
	}
	t.Logf("%d TEST CASES OUT of %d FAILED", failcnt, i-from+1)
}

func breakTestInput(ti string) (string, []string) {
	//fmt.Printf("breaking up %s\n", ti)
	sc := bufio.NewScanner(strings.NewReader(ti))
	sc.Split(bufio.ScanWords)
	out := make([]string, 0, 5)
	inp := bytes.NewBuffer(make([]byte, 0, 20))
	run := bytes.NewBuffer(make([]byte, 0, 20))
	for sc.Scan() {
		token := sc.Text()
		if token == "÷" {
			if run.Len() > 0 {
				out = append(out, run.String())
				run.Reset()
			}
		} else if token == "×" {
			// do nothing
		} else {
			n, _ := strconv.ParseUint(token, 16, 64)
			run.WriteRune(rune(n))
			inp.WriteRune(rune(n))
		}
	}
	//fmt.Printf("input = '%s'\n", inp.String())
	//fmt.Printf("output = %#v\n", out)
	return inp.String(), out
}

func executeSingleTest(t *testing.T, seg *segment.Segmenter, tno int, in string, out []string) bool {
	seg.Init(strings.NewReader(in))
	i := 0
	ok := true
	for seg.Next() {
		if out[i] != seg.Text() {
			t.Errorf("test #%d: '%+q' should be '%+q'", tno, seg.Bytes(), out[i])
			ok = false
		}
		i++
	}
	return ok
}
