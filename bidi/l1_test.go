package bidi

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/unicode/bidi"
)

func TestApplyL1Visual(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		dir        Direction
		end        int // zero means the whole paragraph
		from, to   int
		want       string
	}{
		{"wrapped RTL words", "a אב  גד", LeftToRight, 8, 6, 8, "a בא  "},
		{"RTL isolate", "a\u2067אב  גד\u2069z", LeftToRight, 10, 8, 10, "aבא  "},
		{"nested same direction", "a\u2067אב\u2066cd  ef\u2069גד\u2069z", LeftToRight, 15, 13, 15, "acdבא  "},
		{"isolate closing control", "a\u2067אב  \u2069", LeftToRight, 0, 8, 13, "aבא  "},
		{"separator inside isolate", "a\u2067אב \tגד\u2069z", LeftToRight, 0, 8, 10, "aבא \tדגz"},
		{"crossing isolate boundaries", "a\u2067אב \u2066 \u2069 \u2069z", LeftToRight, 0, 8, 20, "aבא   z"},
		{"multiple siblings", "a\u2067אב \u2069 \u2066 \u2069z", LeftToRight, 0, 8, 20, "aבא   z"},
		{"RTL paragraph", "אב ab  cd", RightToLeft, 9, 7, 9, "  ab בא"},
		{"RTL paragraph LTR isolate", "אב\u2066cd  ef\u2069", RightToLeft, 11, 9, 11, "  cdבא"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			levels := ResolveParagraph(strings.NewReader(tc.text), nil, DefaultDirection(tc.dir))
			if tc.end != 0 {
				levels, _ = levels.Split(uint64(tc.end), false)
			}
			if err := levels.ApplyL1(uint64(tc.from), uint64(tc.to)); err != nil {
				t.Fatal(err)
			}
			ordering := levels.Reorder()
			if got := l1Visual(tc.text, ordering); got != tc.want {
				t.Logf("runs: %v", ordering.Runs)
				t.Fatalf("visual = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyL1RepeatedAndRebased(t *testing.T) {
	const text = "x\u2067אב \u2066cd   ef\u2069גד\u2069z"
	for _, dir := range []Direction{LeftToRight, RightToLeft} {
		original := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(dir))
		untouched := original.Clone()
		for _, rebase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/rebase=%t", dir, rebase), func(t *testing.T) {
				// The line starts within the outer isolate and ends within the
				// nested isolate. Both sides need continuation delimiters.
				_, rest := original.Clone().Split(4, rebase)
				offset := uint64(0)
				if rebase {
					offset = 4
				}
				line, suffix := rest.Split(17-offset, false)
				suffixBefore := suffix.Clone()
				baseline := slices.Collect(line.LogicalSegments())
				reset := func(levels *ResolvedLevels, from, to uint64) {
					t.Helper()
					if err := levels.ApplyL1(from-offset, to-offset); err != nil {
						t.Fatal(err)
					}
				}
				union := line.Clone()
				reset(union, 14, 17)
				reset(line, 15, 17)
				reset(line, 14, 16) // overlapping and in reverse source order
				reset(line, 15, 17) // already reset
				if got, want := l1Visual(text[offset:], line.Clone().Reorder()), l1Visual(text[offset:], union.Clone().Reorder()); got != want {
					t.Fatalf("overlapping resets = %q, union = %q", got, want)
				}
				// Build expected byte directions independently of scrap boundaries.
				var want []Segment
				for _, s := range baseline {
					for b := s.From; b < s.To; b++ {
						direction := s.Dir
						if b >= 14-offset && b < 17-offset {
							direction = dir
						}
						want = append(want, Segment{direction, b, b + 1})
					}
				}
				// Applying L1 must leave the result splittable, and Clone must
				// preserve the reset state without sharing mutable storage.
				left, right := line.Clone().Split(16-offset, false)
				checkSplitCoverage(t, left, want, 4-offset, 16-offset, 0)
				checkSplitCoverage(t, right, want, 16-offset, 17-offset, 0)
				checkSplitCoverage(t, line, want, 4-offset, 17-offset, 0)
				if !reflect.DeepEqual(suffix, suffixBefore) || !reflect.DeepEqual(original, untouched) {
					t.Fatal("resetting or reordering the line modified its suffix or the original")
				}
			})
		}
	}
}

func TestApplyL1MarkupAndMultipleIntervals(t *testing.T) {
	const text = "אב \tגד   "
	markup := func(at uint64) int {
		return map[uint64]int{0: MarkupRLI, uint64(len(text)): MarkupPDI}[at]
	}
	levels := ResolveParagraph(strings.NewReader(text), markup)
	if err := levels.ApplyL1(4, 6); err != nil {
		t.Fatal(err)
	}
	if err := levels.ApplyL1(10, 13); err != nil {
		t.Fatal(err)
	}
	if got := l1Visual(text, levels.Reorder()); got != "בא \tדג   " {
		t.Fatalf("visual = %q", got)
	}
}

func TestReorderNestedVisual(t *testing.T) {
	for _, tc := range []struct {
		text, want string
		dir        Direction
	}{
		{"a\u2067אב\u2066cd\u2069גד\u2069z", "aדגcdבאz", LeftToRight},
		{"אב\u2066cd\u2067גד\u2069ef\u2069וז", "זוcdדגefבא", RightToLeft},
		{"אב 12 גד 34", "34 דג 12 בא", RightToLeft},
		{"a\u2067אב\u2067גד\u2069הו\u2069z", "aוהדגבאz", LeftToRight},
	} {
		t.Run(tc.text, func(t *testing.T) {
			levels := ResolveParagraph(strings.NewReader(tc.text), nil, DefaultDirection(tc.dir))
			if got := l1Visual(tc.text, levels.Reorder()); got != tc.want {
				t.Fatalf("visual = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReorderUAXExample4(t *testing.T) {
	// UAX#9 section 3.4, example 4: nested isolates in an RTL paragraph.
	// TestMode classifies uppercase Latin letters as RTL for the UAX notation.
	const text = "DID YOU SAY ’\u2066he said “\u2067car MEANS CAR\u2069”\u2069‘?"
	const want = "?‘he said “RAC SNAEM car”’ YAS UOY DID"
	levels := ResolveParagraph(strings.NewReader(text), nil, TestMode(true), DefaultDirection(RightToLeft))
	if got := l1Visual(text, levels.Reorder()); got != want {
		t.Fatalf("visual = %q, want %q", got, want)
	}
}

func TestApplyL1IntervalBoundaries(t *testing.T) {
	const text = "a\u2067אב \u2066 \u2069 \u2069z"
	var boundaries []uint64
	for at := range text {
		if at >= 8 && at <= 20 { // whitespace and nested controls
			boundaries = append(boundaries, uint64(at))
		}
	}
	for _, dir := range []Direction{LeftToRight, RightToLeft} {
		original := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(dir))
		baseline := slices.Collect(original.LogicalSegments())
		for i, from := range boundaries {
			for _, to := range boundaries[i:] {
				t.Run(fmt.Sprintf("%s/%d:%d", dir, from, to), func(t *testing.T) {
					line := original.Clone()
					if err := line.ApplyL1(from, to); err != nil {
						t.Fatal(err)
					}
					var want []Segment
					for _, s := range baseline {
						for b := s.From; b < s.To; b++ {
							direction := s.Dir
							if b >= from && b < to {
								direction = dir
							}
							want = append(want, Segment{direction, b, b + 1})
						}
					}
					once := l1Visual(text, line.Clone().Reorder())
					if err := line.ApplyL1(from, to); err != nil {
						t.Fatal(err)
					}
					if twice := l1Visual(text, line.Clone().Reorder()); twice != once {
						t.Fatalf("reset not idempotent: once=%q, twice=%q", once, twice)
					}
					checkSplitCoverage(t, line, want, 0, uint64(len(text)), 0)
				})
			}
		}
	}
}

// These fixtures have no combining marks or shaping; reversing runes is enough
// to inspect their visual order. Real rendering must use the shaped glyphs.
func l1Visual(text string, ordering *Ordering) string {
	var out strings.Builder
	for _, run := range ordering.Runs {
		for s := range run.Segments(run.IsOpposite(LeftToRight)) {
			part := []rune(text[s.From:s.To])
			if s.Dir == RightToLeft {
				slices.Reverse(part)
			}
			for _, r := range part {
				if r < '\u2066' || r > '\u2069' {
					out.WriteRune(r)
				}
			}
		}
	}
	return out.String()
}

func TestApplyL1Validation(t *testing.T) {
	original := ResolveParagraph(strings.NewReader("abc def"), nil)
	_, line := original.Split(4, false)
	for _, span := range [][2]uint64{{5, 4}, {3, 5}, {4, 8}, {8, 8}, {0, 0}, {1 << 32, 1<<32 + 1}} {
		before := line.Clone()
		if err := line.ApplyL1(span[0], span[1]); err == nil {
			t.Errorf("invalid interval %v accepted", span)
		}
		if !reflect.DeepEqual(line, before) {
			t.Fatalf("invalid interval %v modified levels", span)
		}
	}
	for _, at := range []uint64{4, 5, 7} {
		before := line.Clone()
		if err := line.ApplyL1(at, at); err != nil || !reflect.DeepEqual(line, before) {
			t.Fatalf("empty interval at %d: err=%v or levels changed", at, err)
		}
	}
	var empty ResolvedLevels
	if err := empty.ApplyL1(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := empty.ApplyL1(0, 1); err == nil {
		t.Fatal("nonempty interval on empty levels accepted")
	}
}

func TestApplyL1Controls(t *testing.T) {
	for _, dir := range []Direction{LeftToRight, RightToLeft} {
		for _, text := range []string{"\u2067\u2069", " \u2067\u2066\u2069\u2069 "} {
			t.Run(fmt.Sprintf("%s/%q", dir, text), func(t *testing.T) {
				levels := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(dir))
				baseline := slices.Collect(levels.LogicalSegments())
				if err := levels.ApplyL1(0, uint64(len(text))); err != nil {
					t.Fatal(err)
				}
				for i := range baseline {
					baseline[i].Dir = dir
				}
				if got := slices.Collect(levels.LogicalSegments()); !slices.Equal(got, baseline) {
					t.Fatalf("controls exposed as text: %v, want %v", got, baseline)
				}
				// Check source controls retain identity, but now have base direction.
				var controls []bidi.Class
				for _, s := range levels.scraps {
					if isisolate(s) && s.l < s.r {
						controls = append(controls, s.bidiclz)
						if directionFromBidiClass(s, dir) != dir {
							t.Fatal("control not reset to paragraph direction")
						}
					}
				}
				if len(controls) != strings.Count(text, "\u2067")+strings.Count(text, "\u2066")+strings.Count(text, "\u2069") {
					t.Fatal("lost control identity")
				}
				checkSplitCoverage(t, levels, baseline, 0, uint64(len(text)), 0)
			})
		}
	}
}
