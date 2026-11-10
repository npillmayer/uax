package bidi

import (
	"reflect"
	"strings"
	"testing"
)

func TestSoftHyphenDoesNotSplitLatinInRTL(t *testing.T) {
	for _, text := range []string{"a\u00adb", "é\u00ad\u00adß", "\u00adab", "ab\u00ad"} {
		levels := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(RightToLeft))
		var logical []Segment
		for s := range levels.LogicalSegments() {
			logical = append(logical, s)
		}
		want := []Segment{{Dir: LeftToRight, From: 0, To: uint64(len(text))}}
		if !reflect.DeepEqual(logical, want) {
			t.Fatalf("%q logical %v want %v", text, logical, want)
		}
		var visual []Segment
		for _, run := range levels.Reorder().Runs {
			for s := range run.Segments(run.IsOpposite(LeftToRight)) {
				visual = append(visual, s)
			}
		}
		if !reflect.DeepEqual(visual, want) {
			t.Fatalf("%q visual %v want %v", text, visual, want)
		}
	}
}

func TestBoundaryNeutralsPreserveVisibleOrdering(t *testing.T) {
	for _, dir := range []Direction{LeftToRight, RightToLeft} {
		for _, text := range []string{"\u00ad", "\u00ad\u00ad", "a\u00ad\u0301b", "אב\u00adגד", "1\u00ad2,34", "a (\u00adb) אב", "a\u2067\u00adאב\u00ad\u2069\u00adb", "a\u200bb", "a\u2060b"} {
			clean := strings.NewReplacer("\u00ad", "", "\u200b", "", "\u2060", "")
			visual := func(text string) string {
				levels := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(dir))
				var out strings.Builder
				for _, run := range levels.Reorder().Runs {
					for seg := range run.Segments(run.IsOpposite(LeftToRight)) {
						chars := []rune(text[seg.From:seg.To])
						if seg.Dir == RightToLeft {
							for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
								chars[i], chars[j] = chars[j], chars[i]
							}
						}
						out.WriteString(clean.Replace(string(chars)))
					}
				}
				return out.String()
			}
			if got, want := visual(text), visual(clean.Replace(text)); got != want {
				t.Errorf("%s %q: visual %q want %q", dir, text, got, want)
			}
		}
	}
}
