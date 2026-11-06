package bidi

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestSplitCharacterBoundaries(t *testing.T) {
	for _, text := range []string{
		"ab cd ef", "אב 12 גד 34", "\u2066\u2069", "אב \u2066ab cd\u2069 גד",
		"a \u2067אב\u2069 middle \u2067גד\u2069 z",
		"ab \u2067אב גד\u2069 ef",
		"a \u2067אב \u2066cd ef\u2069 גד\u2069 z",
	} {
		for _, dir := range []Direction{LeftToRight, RightToLeft} {
			original := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(dir))
			baseline := slices.Collect(original.LogicalSegments())
			var cuts []uint64
			for at := range text {
				cuts = append(cuts, uint64(at))
			}
			cuts = append(cuts, uint64(len(text)))
			for _, at := range cuts {
				for _, rebase := range []bool{false, true} {
					t.Run(fmt.Sprintf("%q/%s/%d/rebase=%t", text, dir, at, rebase), func(t *testing.T) {
						prefix, suffix := original.Clone().Split(at, rebase)
						if prefix.embedding != dir || suffix.embedding != dir {
							t.Fatal("split changed embedding direction")
						}
						// Consume the prefix before inspecting the suffix, to catch sharing
						// between the two results as well as incorrect interval bounds.
						checkSplitCoverage(t, prefix, baseline, 0, at, 0)
						offset := uint64(0)
						if rebase {
							offset = at
						}
						checkSplitCoverage(t, suffix, baseline, at, uint64(len(text)), offset)
					})
				}
			}
			if !slices.Equal(baseline, slices.Collect(original.LogicalSegments())) {
				t.Fatal("splitting a clone changed the original")
			}
		}
	}
}

// Compare logical and visual output to the original paragraph clipped to the
// line. Source controls may appear in visual output but must not be duplicated
// or escape the line. Every text byte must appear exactly once in both views.
func checkSplitCoverage(t *testing.T, part *ResolvedLevels, baseline []Segment, lo, hi, offset uint64) {
	t.Helper()
	want := make(map[uint64]Direction)
	for _, s := range baseline {
		from, to := s.From, s.To
		if from < lo {
			from = lo
		}
		if to > hi {
			to = hi
		}
		for b := from; b < to; b++ {
			want[b-offset] = s.Dir
		}
	}
	check := func(segments []Segment, visual bool) {
		t.Helper()
		got := make(map[uint64]Direction)
		seen := make(map[uint64]bool)
		for _, s := range segments {
			if s.From < lo-offset || s.To > hi-offset || s.From >= s.To {
				t.Fatalf("visual=%t: interval %v outside [%d,%d)", visual, s, lo-offset, hi-offset)
			}
			for b := s.From; b < s.To; b++ {
				if seen[b] {
					t.Fatalf("visual=%t: duplicated byte %d in %v", visual, b, segments)
				}
				seen[b] = true
				if _, text := want[b]; !visual || text {
					got[b] = s.Dir
				}
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("visual=%t: source coverage/direction mismatch: got %v, want %v; segments=%v", visual, got, want, segments)
		}
	}
	check(slices.Collect(part.LogicalSegments()), false)
	check(splitTestSegments(part), true)
}

func TestSplitRepeated(t *testing.T) {
	const text = "a \u2067אב \u2066cd ef\u2069 גד\u2069 z"
	for _, rebase := range []bool{false, true} {
		t.Run(fmt.Sprintf("rebase=%t", rebase), func(t *testing.T) {
			original := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(RightToLeft))
			baseline := slices.Collect(original.LogicalSegments())
			rest := original.Clone()
			start := uint64(0)
			for _, at := range []uint64{10, 16, 24, uint64(len(text))} {
				cut, offset := at, uint64(0)
				if rebase {
					cut, offset = at-start, start
				}
				line, suffix := rest.Split(cut, rebase)
				checkSplitCoverage(t, line, baseline, start, at, offset)
				rest, start = suffix, at
			}
			left, right := rest.Split(0, rebase)
			if len(left.Reorder().Runs) != 0 || len(right.Reorder().Runs) != 0 {
				t.Fatal("repeated split of an empty suffix yielded runs")
			}
		})
	}
}

func TestSplitMarkupBoundaries(t *testing.T) {
	const text = "aאבcdגדz"
	for name, marks := range map[string]map[uint64]int{
		"nested":   {1: MarkupRLI, 5: MarkupLRI, 7: MarkupPDI, 11: MarkupPDI},
		"siblings": {1: MarkupRLI, 5: MarkupPDILRI, 7: MarkupPDIRLI, 11: MarkupPDI},
	} {
		for _, dir := range []Direction{LeftToRight, RightToLeft} {
			original := ResolveParagraph(strings.NewReader(text), func(at uint64) int {
				return marks[at]
			}, DefaultDirection(dir))
			baseline := slices.Collect(original.LogicalSegments())
			for at := range text {
				for _, rebase := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%d/rebase=%t", name, dir, at, rebase), func(t *testing.T) {
						left, right := original.Clone().Split(uint64(at), rebase)
						checkSplitCoverage(t, left, baseline, 0, uint64(at), 0)
						offset := uint64(0)
						if rebase {
							offset = uint64(at)
						}
						checkSplitCoverage(t, right, baseline, uint64(at), uint64(len(text)), offset)
					})
				}
			}
		}
	}
}

func TestSplitOutsideRange(t *testing.T) {
	original := ResolveParagraph(strings.NewReader("abc def"), nil, DefaultDirection(RightToLeft))
	_, rest := original.Split(4, false)
	for _, at := range []uint64{0, 3, 4, 7, 8, 1<<32 + 5} {
		for _, rebase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/rebase=%t", at, rebase), func(t *testing.T) {
				left, right := rest.Clone().Split(at, rebase)
				if left.embedding != RightToLeft || right.embedding != RightToLeft {
					t.Fatal("boundary split lost embedding")
				}
				if at <= 4 {
					if len(left.Reorder().Runs) != 0 {
						t.Fatal("cut before start produced nonempty prefix")
					}
					from, to := uint64(4), uint64(7)
					if rebase {
						from, to = from-at, to-at
					}
					if got := slices.Collect(right.LogicalSegments()); !slices.Equal(got, []Segment{{LeftToRight, from, to}}) {
						t.Fatalf("unexpected suffix: %v", got)
					}
				} else if len(right.Reorder().Runs) != 0 {
					t.Fatal("cut beyond end produced nonempty suffix")
				}
			})
		}
	}
}
