package grapheme

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/npillmayer/uax/internal/testdata"
	"github.com/npillmayer/uax/internal/tracing"
	"github.com/npillmayer/uax/segment"
)

func TestGraphemeClasses(t *testing.T) {
	tracing.SetTestingLog(t)
	c1 := LClass
	if c1.String() != "LClass" {
		t.Errorf("String(LClass) should be 'LClass', is %s", c1)
	}
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
	tracing.SetTestingLog(t)

	onGraphemes := NewBreaker(1)
	input := bytes.NewReader([]byte("개=Hang Syllable GAE"))
	seg := segment.NewSegmenter(onGraphemes)
	seg.Init(input)
	seg.Next()
	t.Logf("Next() = %s\n", seg.Text())
	if seg.Err() != nil {
		t.Errorf("segmenter.Next() failed with error: %s", seg.Err())
	}
	if seg.Text() != "개" {
		t.Errorf("expected first grapheme of string to be '개' (Hang GAE), is '%v'", seg.Text())
	}
}

func TestGraphemes2(t *testing.T) {
	tracing.SetTestingLog(t)

	onGraphemes := NewBreaker(1)
	input := bytes.NewReader([]byte("Hello\tWorld!"))
	seg := segment.NewSegmenter(onGraphemes)
	seg.BreakOnZero(true, false)
	seg.Init(input)
	output := ""
	for seg.Next() {
		t.Logf("Next() = %s\n", seg.Text())
		output += "_" + seg.Text()
	}
	if seg.Err() != nil {
		t.Errorf("segmenter.Next() failed with error: %s", seg.Err())
	}
	if output != "_H_e_l_l_o_\t_W_o_r_l_d_!" {
		t.Errorf("expected grapheme for every char pos, have %s", output)
	}
}

func TestGraphemesTestFile(t *testing.T) {
	tracing.SetTestingLog(t)

	onGraphemes := NewBreaker(5)
	seg := segment.NewSegmenter(onGraphemes)
	//seg.BreakOnZero(true, false)
	//failcnt, i, from, to := 0, 0, 1, 1000
	failcnt, i, from, to := 0, 0, 1, 1000
	scan := bufio.NewScanner(bytes.NewReader(testdata.GraphemeBreakTest))
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
			//TC().Infof("#######################################################")
			tracing.Infof(comment)
			in, out := breakTestInput(testInput)
			success := executeSingleTest(t, seg, i, in, out)
			_, shouldFail := knownFailure[testInput]
			shouldSucceed := !shouldFail
			if success != shouldSucceed {
				failcnt++
				if shouldFail {
					t.Logf("expected %q to fail, but succeeded", testInput)
				}
			}
		}
		if i >= to {
			break
		}
	}
	if err := scan.Err(); err != nil {
		tracing.Infof("reading input: %v", err)
	}
	if failcnt > 0 {
		t.Errorf("%d TEST CASES OUT of %d FAILED", failcnt, i-from+1)
	}
}

var knownFailure = map[string]struct{}{
	"÷ 0600 × 0020 ÷\t":         {},
	"÷ 0600 × 1F1E6 ÷\t":        {},
	"÷ 0600 × 0600 ÷\t":         {},
	"÷ 0600 × 1100 ÷\t":         {},
	"÷ 0600 × 1160 ÷\t":         {},
	"÷ 0600 × 11A8 ÷\t":         {},
	"÷ 0600 × AC00 ÷\t":         {},
	"÷ 0600 × AC01 ÷\t":         {},
	"÷ 0600 × 231A ÷\t":         {},
	"÷ 0600 × 0378 ÷\t":         {},
	"÷ D800 ÷ 0308 ÷ 0020 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 000D ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 000A ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 0001 ÷\t":  {},
	"÷ D800 ÷ 034F ÷\t":         {},
	"÷ D800 ÷ 0308 × 034F ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 1F1E6 ÷\t": {},
	"÷ D800 ÷ 0308 ÷ 0600 ÷\t":  {},
	"÷ D800 ÷ 0903 ÷\t":         {},
	"÷ D800 ÷ 0308 × 0903 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 1100 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 1160 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 11A8 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ AC00 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ AC01 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 231A ÷\t":  {},
	"÷ D800 ÷ 0300 ÷\t":         {},
	"÷ D800 ÷ 0308 × 0300 ÷\t":  {},
	"÷ D800 ÷ 200D ÷\t":         {},
	"÷ D800 ÷ 0308 × 200D ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ 0378 ÷\t":  {},
	"÷ D800 ÷ 0308 ÷ D800 ÷\t":  {},
	"÷ 0061 ÷ 0600 × 0062 ÷\t":  {},
}

func breakTestInput(ti string) (string, []string) {
	//fmt.Printf("breaking up %s\n", ti)
	sc := bufio.NewScanner(strings.NewReader(ti))
	sc.Split(bufio.ScanWords)
	out := make([]string, 0, 10)
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
	tracing.Infof("expecting %v", ost(out))
	seg.Init(strings.NewReader(in))
	i := 0
	ok := true
	for seg.Next() {
		if i >= len(out) {
			t.Logf("broken lexemes longer than expected output")
		} else if out[i] != seg.Text() {
			p0, p1 := seg.Penalties()
			t.Logf("test #%d: penalties = %d|%d", tno, p0, p1)
			t.Logf("test #%d: '%x' should be '%x'", tno, string(seg.Bytes()), out[i])
			ok = false
		}
		i++
	}
	return ok
}

func ost(out []string) string {
	s := ""
	first := true
	for _, o := range out {
		if first {
			first = false
		} else {
			s += "-"
		}
		s += "[" + o + "]"
	}
	return s
}
