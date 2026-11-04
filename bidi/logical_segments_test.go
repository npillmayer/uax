package bidi

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestLogicalSegments(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		opts   []Option
		markup OutOfLineBidiMarkup
		want   []Segment
	}{
		{name: "empty"},
		{
			name: "style example", text: "one two THREE FOUR FIVE",
			opts: []Option{TestMode(true)},
			want: []Segment{{LeftToRight, 0, 8}, {RightToLeft, 8, 23}},
		},
		{
			name: "UTF-8", text: "Go אבג!",
			want: []Segment{{LeftToRight, 0, 3}, {RightToLeft, 3, 9}, {LeftToRight, 9, 10}},
		},
		{
			name: "RTL punctuation", text: "אבג!",
			opts: []Option{DefaultDirection(RightToLeft)},
			want: []Segment{{RightToLeft, 0, 7}},
		},
		{
			name: "European numbers in RTL paragraph", text: "אבג 123 דה",
			opts: []Option{DefaultDirection(RightToLeft)},
			want: []Segment{{RightToLeft, 0, 7}, {LeftToRight, 7, 10}, {RightToLeft, 10, 15}},
		},
		{
			name: "Arabic numbers", text: "اب ١٢ ج",
			want: []Segment{{RightToLeft, 0, 5}, {LeftToRight, 5, 9}, {RightToLeft, 9, 12}},
		},
		{
			name: "nested isolates", text: "a\u2067אב\u2066cd\u2069גד\u2069z",
			want: []Segment{
				{LeftToRight, 0, 1}, {RightToLeft, 4, 8}, {LeftToRight, 11, 13},
				{RightToLeft, 16, 20}, {LeftToRight, 23, 24},
			},
		},
		{
			name: "same direction isolate", text: "a\u2066b\u2069c",
			want: []Segment{{LeftToRight, 0, 1}, {LeftToRight, 4, 5}, {LeftToRight, 8, 9}},
		},
		{
			name: "isolate covering paragraph", text: "\u2067אב\u2069",
			want: []Segment{{RightToLeft, 3, 7}},
		},
		{
			name: "empty isolate", text: "a\u2067\u2069b",
			want: []Segment{{LeftToRight, 0, 1}, {LeftToRight, 7, 8}},
		},
		{
			name: "sibling isolates", text: "a\u2067אב\u2069b\u2067גד\u2069c",
			want: []Segment{
				{LeftToRight, 0, 1}, {RightToLeft, 4, 8}, {LeftToRight, 11, 12},
				{RightToLeft, 15, 19}, {LeftToRight, 22, 23},
			},
		},
		{
			name: "nested external markup", text: "aאבcdגדz",
			markup: func(pos uint64) int {
				return map[uint64]int{1: MarkupRLI, 5: MarkupLRI, 7: MarkupPDI, 11: MarkupPDI}[pos]
			},
			want: []Segment{
				{LeftToRight, 0, 1}, {RightToLeft, 1, 5}, {LeftToRight, 5, 7},
				{RightToLeft, 7, 11}, {LeftToRight, 11, 12},
			},
		},
		{
			name: "same direction external markup", text: "abc",
			markup: func(pos uint64) int {
				return map[uint64]int{1: MarkupLRI, 2: MarkupPDI}[pos]
			},
			want: []Segment{{LeftToRight, 0, 1}, {LeftToRight, 1, 2}, {LeftToRight, 2, 3}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levels := ResolveParagraph(strings.NewReader(tt.text), tt.markup, tt.opts...)
			// Resolve independently so nested backing slices cannot alias the snapshot.
			untouched := ResolveParagraph(strings.NewReader(tt.text), tt.markup, tt.opts...)
			seq := levels.LogicalSegments()
			for pass := 0; pass < 2; pass++ {
				if got := slices.Collect(seq); !slices.Equal(got, tt.want) {
					t.Fatalf("pass %d: got %v, want %v", pass, got, tt.want)
				}
			}
			if !reflect.DeepEqual(levels, untouched) {
				t.Fatal("logical iteration modified the resolved levels")
			}
			if got, want := levels.Reorder(), untouched.Reorder(); !reflect.DeepEqual(got, want) {
				t.Fatalf("logical iteration affected subsequent reordering: got %v, want %v", got, want)
			}
		})
	}
}

func TestLogicalSegmentsZeroValue(t *testing.T) {
	var levels ResolvedLevels
	for segment := range levels.LogicalSegments() {
		t.Fatalf("zero value yielded %v", segment)
	}
}

func TestLogicalSegmentsEarlyExitAndIndependentTraversals(t *testing.T) {
	levels := ResolveParagraph(strings.NewReader("a\u2067אב\u2066cd\u2069גד\u2069z"), nil)
	want := []Segment{
		{LeftToRight, 0, 1}, {RightToLeft, 4, 8}, {LeftToRight, 11, 13},
		{RightToLeft, 16, 20}, {LeftToRight, 23, 24},
	}
	seq := levels.LogicalSegments()
	for stop := 1; stop <= len(want); stop++ {
		var prefix []Segment
		for segment := range seq {
			prefix = append(prefix, segment)
			if len(prefix) == stop {
				break // No yield may follow, even when stopping inside nested isolates.
			}
		}
		if !slices.Equal(prefix, want[:stop]) {
			t.Fatalf("prefix = %v, want %v", prefix, want[:stop])
		}
	}
	var outer []Segment
	for segment := range seq {
		outer = append(outer, segment)
		if inner := slices.Collect(seq); !slices.Equal(inner, want) {
			t.Fatalf("nested traversal = %v, want %v", inner, want)
		}
	}
	if !slices.Equal(outer, want) {
		t.Fatalf("outer traversal = %v, want %v", outer, want)
	}
}

func TestLogicalSegmentsBeforeSplit(t *testing.T) {
	const text = "one two THREE FOUR FIVE"
	levels := ResolveParagraph(strings.NewReader(text), nil, TestMode(true))
	_ = slices.Collect(levels.LogicalSegments())
	first, rest := levels.Split(14, false)
	untouched := ResolveParagraph(strings.NewReader(text), nil, TestMode(true))
	wantFirst, wantRest := untouched.Split(14, false)
	if !reflect.DeepEqual(first, wantFirst) || !reflect.DeepEqual(rest, wantRest) {
		t.Fatal("logical iteration affected subsequent line splitting")
	}
}
