package bidi

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestResolvedLevelsClone(t *testing.T) {
	const text = "DID YOU SAY ’>he said “<car MEANS CAR=”=‘?"
	resolve := func() *ResolvedLevels {
		return ResolveParagraph(strings.NewReader(text), nil, TestMode(true), DefaultDirection(RightToLeft))
	}
	operations := []struct {
		name string
		run  func(*ResolvedLevels)
	}{
		{"reorder", func(levels *ResolvedLevels) { levels.Reorder() }},
		{"split nested isolate", func(levels *ResolvedLevels) {
			levels.Split(uint64(strings.Index(text, "MEANS")+2), false)
		}},
		{"split and rebase", func(levels *ResolvedLevels) {
			levels.Split(uint64(strings.Index(text, "MEANS")+2), true)
		}},
	}
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			original := resolve()
			baseline := resolve() // An independent reference, not another clone.
			cloned := original.Clone()
			other := original.Clone()
			if cloned == original || !reflect.DeepEqual(cloned, baseline) {
				t.Fatal("clone must be a distinct, equal copy, including embedding and context")
			}
			wantLogical := slices.Collect(original.LogicalSegments())
			op.run(cloned)
			if reflect.DeepEqual(cloned, baseline) {
				t.Fatal("test operation did not exercise mutation")
			}
			if !reflect.DeepEqual(original, baseline) || !reflect.DeepEqual(other, baseline) {
				t.Fatal("mutating a clone changed the original or another clone")
			}
			if got := slices.Collect(original.LogicalSegments()); !slices.Equal(got, wantLogical) {
				t.Fatalf("original logical segments changed: got %v, want %v", got, wantLogical)
			}
			// Independence also holds when the original is consumed afterward.
			op.run(original)
			if !reflect.DeepEqual(other, baseline) {
				t.Fatal("mutating the original changed an existing clone")
			}
			if got, want := other.Reorder(), baseline.Reorder(); !reflect.DeepEqual(got, want) {
				t.Fatalf("clone reordering differs from fresh resolution: got %v, want %v", got, want)
			}
		})
	}
}

func TestResolvedLevelsCloneEmpty(t *testing.T) {
	for _, original := range []*ResolvedLevels{
		{},
		{scraps: []scrap{}},
		ResolveParagraph(strings.NewReader(""), nil),
		ResolveParagraph(strings.NewReader("\u2067\u2069"), nil),
	} {
		cloned := original.Clone()
		if cloned == original || !reflect.DeepEqual(cloned, original) {
			t.Fatal("empty clone must be a distinct, equal copy")
		}
		if got := slices.Collect(cloned.LogicalSegments()); len(got) != 0 {
			t.Fatalf("empty clone yielded %v", got)
		}
	}
}
