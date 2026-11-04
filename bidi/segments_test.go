package bidi

import (
	"slices"
	"testing"
)

func TestSegments(t *testing.T) {
	tests := []struct {
		name    string
		run     Run
		reverse bool
		want    []Segment
	}{
		{name: "empty"},
		{name: "empty reverse", reverse: true},
		{
			name: "single LTR",
			run:  Run{Dir: LeftToRight, scraps: []scrap{{l: 3, r: 9}}},
			want: []Segment{{Dir: LeftToRight, From: 3, To: 9}},
		},
		{
			name:    "single RTL reverse",
			run:     Run{Dir: RightToLeft, scraps: []scrap{{l: 3, r: 9}}},
			reverse: true,
			want:    []Segment{{Dir: RightToLeft, From: 3, To: 9}},
		},
		{
			name: "noncontiguous forward",
			run: Run{Dir: RightToLeft, scraps: []scrap{
				{l: 20, r: 26}, {l: 3, r: 9}, {l: 12, r: 16},
			}},
			want: []Segment{
				{Dir: RightToLeft, From: 20, To: 26},
				{Dir: RightToLeft, From: 3, To: 9},
				{Dir: RightToLeft, From: 12, To: 16},
			},
		},
		{
			name: "noncontiguous reverse",
			run: Run{Dir: RightToLeft, scraps: []scrap{
				{l: 20, r: 26}, {l: 3, r: 9}, {l: 12, r: 16},
			}},
			reverse: true,
			want: []Segment{
				{Dir: RightToLeft, From: 12, To: 16},
				{Dir: RightToLeft, From: 3, To: 9},
				{Dir: RightToLeft, From: 20, To: 26},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Collect(tt.run.Segments(false))
			seq := tt.run.Segments(tt.reverse)
			for pass := 0; pass < 2; pass++ {
				if got := slices.Collect(seq); !slices.Equal(got, tt.want) {
					t.Fatalf("pass %d: got %v, want %v", pass, got, tt.want)
				}
			}
			if after := slices.Collect(tt.run.Segments(false)); !slices.Equal(after, before) {
				t.Fatalf("iteration changed the run: got %v, want %v", after, before)
			}
		})
	}
}

func TestSegmentsEarlyExitAndIndependentTraversals(t *testing.T) {
	run := Run{Dir: RightToLeft, scraps: []scrap{{l: 3, r: 9}, {l: 12, r: 16}}}
	want := []Segment{
		{Dir: RightToLeft, From: 12, To: 16},
		{Dir: RightToLeft, From: 3, To: 9},
	}
	seq := run.Segments(true)
	for segment := range seq {
		if segment != want[0] {
			t.Fatalf("first segment = %v, want %v", segment, want[0])
		}
		break // A yield after this break would panic.
	}
	var outer []Segment
	for segment := range seq {
		outer = append(outer, segment)
		if inner := slices.Collect(seq); !slices.Equal(inner, want) {
			t.Fatalf("nested traversal = %v, want %v", inner, want)
		}
	}
	if !slices.Equal(outer, want) {
		t.Fatalf("traversal after early exit = %v, want %v", outer, want)
	}
}
