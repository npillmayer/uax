package uax14_test

import (
	"strings"
	"testing"

	"github.com/npillmayer/uax/internal/testdata"
	"github.com/npillmayer/uax/internal/tracing"
	"github.com/npillmayer/uax/internal/ucdparse"
	"github.com/npillmayer/uax/segment"
	"github.com/npillmayer/uax/uax14"
)

func TestSimpleLineWrap(t *testing.T) {
	tracing.SetTestingLog(t)
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
	tracing.SetTestingLog(t)

	file, err := testdata.UCDReader("auxiliary/LineBreakTest.txt")
	if err != nil {
		t.Fatal(err)
	}

	linewrap := uax14.NewLineWrap()
	seg := segment.NewSegmenter(linewrap)
	tf := ucdparse.OpenTestReader(file)
	defer tf.Close()

	//failcnt, i, from, to := 0, 0, 6263, 7000
	failcnt, i, from, to := 0, 0, 0, 7000
	for tf.Scan() {
		i++
		if i >= from {
			t.Log(tf.Comment())
			testInput := tf.Text()
			in, out := ucdparse.BreakTestInput(testInput)

			success := executeSingleTest(t, seg, i, in, out)
			_, shouldFail := knownFailure[testInput]
			shouldSucceed := !shouldFail
			if success != shouldSucceed {
				failcnt++
				t.Logf("test #%d failed", i)
				if shouldFail {
					t.Logf("expected %q to fail, but succeeded", testInput)
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

	if failcnt > 0 {
		t.Errorf("%d TEST CASES OUT of %d FAILED", failcnt, i-from+1)
	}
	t.Logf("%d TEST CASES IGNORED", len(knownFailure))
}

var knownFailure = map[string]struct{}{
	"× 0023 ÷ 2329 ÷\t":         {},
	"× 0023 × 0308 ÷ 2329 ÷\t":  {},
	"× 2014 × 2024 ÷\t":         {},
	"× 2014 × 0308 × 2024 ÷\t":  {},
	"× 0009 × 2024 ÷\t":         {},
	"× 0009 × 0308 × 2024 ÷\t":  {},
	"× 000B ÷ 0308 ÷ 2329 ÷\t":  {},
	"× 007D × 2024 ÷\t":         {},
	"× 007D × 0308 × 2024 ÷\t":  {},
	"× 000D ÷ 0308 ÷ 2329 ÷\t":  {},
	"× 05D0 ÷ 2329 ÷\t":         {},
	"× 05D0 × 0308 ÷ 2329 ÷\t":  {},
	"× 002D × 2024 ÷\t":         {},
	"× 002D × 0308 × 2024 ÷\t":  {},
	"× 002C × 2024 ÷\t":         {},
	"× 002C × 0308 × 2024 ÷\t":  {},
	"× 000A ÷ 0308 ÷ 2329 ÷\t":  {},
	"× 0085 ÷ 0308 ÷ 2329 ÷\t":  {},
	"× 17D6 × 2024 ÷\t":         {},
	"× 17D6 × 0308 × 2024 ÷\t":  {},
	"× 0030 ÷ 2329 ÷\t":         {},
	"× 0030 × 0308 ÷ 2329 ÷\t":  {},
	"× 0025 × 2024 ÷\t":         {},
	"× 0025 × 0308 × 2024 ÷\t":  {},
	"× 0024 × 2024 ÷\t":         {},
	"× 0024 × 0308 × 2024 ÷\t":  {},
	"× 0020 ÷ 0308 ÷ 2329 ÷\t":  {},
	"× 002F × 2024 ÷\t":         {},
	"× 002F × 0308 × 2024 ÷\t":  {},
	"× 200B ÷ 0308 ÷ 2329 ÷\t":  {},
	"× 1F1E6 × 2024 ÷\t":        {},
	"× 1F1E6 × 0308 × 2024 ÷\t": {},
	"× 0029 × 2024 ÷\t":         {},
	"× 0029 × 0308 × 2024 ÷\t":  {},
	"× 0001 ÷ 2329 ÷\t":         {},
	"× 0001 × 0308 ÷ 2329 ÷\t":  {},
	"× 200D × 2014 ÷\t":         {},
	"× 200D × 00B4 ÷\t":         {},
	"× 200D × FFFC ÷\t":         {},
	"× 200D × AC00 ÷\t":         {},
	"× 200D × AC01 ÷\t":         {},
	"× 200D × 231A ÷\t":         {},
	"× 200D × 1100 ÷\t":         {},
	"× 200D × 11A8 ÷\t":         {},
	"× 200D × 1160 ÷\t":         {},
	"× 200D × 0308 ÷ 2329 ÷\t":  {},
	"× 200D × 1F1E6 ÷\t":        {},
	"× 200D × 261D ÷\t":         {},
	"× 200D × 1F3FB ÷\t":        {},
	"× 00A7 ÷ 2329 ÷\t":         {},
	"× 00A7 × 0308 ÷ 2329 ÷\t":  {},
	"× 50005 ÷ 2329 ÷\t":        {},
	"× 50005 × 0308 ÷ 2329 ÷\t": {},
}

func executeSingleTest(t *testing.T, seg *segment.Segmenter, tno int, in string, out []string) bool {
	seg.Init(strings.NewReader(in))
	i := 0
	ok := true
	for seg.Next() {
		if len(out) <= i {
			t.Logf("test #%d: number of segments too large: %d > %d", tno, i+1, len(out))
			ok = false
		} else if out[i] != seg.Text() {
			t.Logf("test #%d: '%x' should be '%x'", tno, seg.Bytes(), out[i])
			ok = false
		}
		i++
	}
	return ok
}
