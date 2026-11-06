package bidi

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func splitTestSegments(rl *ResolvedLevels) []Segment {
	var out []Segment
	for _, r := range rl.Reorder().Runs {
		for s := range r.Segments(r.IsOpposite(LeftToRight)) {
			out = append(out, s)
		}
	}
	return out
}

func TestSplitPreservesDirection(t *testing.T) {
	for _, text := range []string{"אב ab גד ef", "אב 12 גד 34", "אב \u2066ab cd\u2069 גד"} {
		t.Run(text, func(t *testing.T) {
			original := ResolveParagraph(strings.NewReader(text), nil, DefaultDirection(RightToLeft))
			at := uint64(strings.Index(text, " ") + 1)
			p, q := original.Clone().Split(at, false)
			t.Logf("embedding original=%v prefix=%v suffix=%v", original.embedding, p.embedding, q.embedding)
			restored := q.Clone()
			restored.embedding = original.embedding
			got, want := splitTestSegments(q), splitTestSegments(restored)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("suffix ordering changes when embedding restored: got=%v restored=%v", got, want)
			}
			if p.embedding != original.embedding || q.embedding != original.embedding {
				t.Error("split lost embedding direction")
			}
		})
	}
}

func TestSplitPartitionsIsolates(t *testing.T) {
	for _, tc := range []struct{ name, text, cut string }{
		{"plain", "ab cd ef", "cd"},
		{"inside isolate", "ab \u2067אב גד\u2069 ef", "גד"},
		{"between isolates", "a \u2067אב\u2069 middle \u2067גד\u2069 z", "middle"},
		{"nested isolate", "a \u2067אב \u2066cd ef\u2069 גד\u2069 z", "ef"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("panic: %v", p)
				}
			}()
			original := ResolveParagraph(strings.NewReader(tc.text), nil)
			baseline := slices.Collect(original.LogicalSegments())
			at := uint64(strings.Index(tc.text, tc.cut))
			p, q := original.Clone().Split(at, false)
			for i, part := range []*ResolvedLevels{p, q} {
				lo, hi := uint64(0), at
				if i == 1 {
					lo, hi = at, uint64(len(tc.text))
				}
				got := slices.Collect(part.LogicalSegments())
				t.Logf("part %d range [%d,%d) segments=%v", i, lo, hi, got)
				want := map[uint64]Direction{}
				actual := map[uint64]Direction{}
				for _, s := range baseline {
					from, to := s.From, s.To
					if from < lo {
						from = lo
					}
					if to > hi {
						to = hi
					}
					for b := from; b < to; b++ {
						want[b] = s.Dir
					}
				}
				for _, s := range got {
					for b := s.From; b < s.To; b++ {
						if _, ok := actual[b]; ok {
							t.Errorf("duplicate source byte %d", b)
						}
						actual[b] = s.Dir
					}
				}
				if !reflect.DeepEqual(want, actual) {
					t.Errorf("part %d does not preserve clipped source coverage/direction", i)
				}
				for _, s := range splitTestSegments(part.Clone()) {
					if s.From < lo || s.To > hi {
						t.Errorf("reordered segment %v outside line [%d,%d)", s, lo, hi)
					}
				}
			}
			if !slices.Equal(baseline, slices.Collect(original.LogicalSegments())) {
				t.Error("clone operation mutated original")
			}
		})
	}
}

func TestSplitEmptyAndBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		at         uint64
		again      bool
	}{
		{"zero value", "", 0, false},
		{"resolved empty", "", 0, false},
		{"paragraph start", "abc", 0, false},
		{"paragraph end", "abc", 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("panic: %v", p)
				}
			}()
			r := ResolveParagraph(strings.NewReader(tc.text), nil)
			if tc.name == "zero value" {
				r = &ResolvedLevels{}
			}
			p, q := r.Clone().Split(tc.at, false)
			t.Logf("prefix=%v suffix=%v", slices.Collect(p.LogicalSegments()), slices.Collect(q.LogicalSegments()))
			if tc.again {
				q.Split(tc.at, false)
			}
		})
	}
}
