package bidi_test

import (
	"fmt"
	"strings"

	"github.com/npillmayer/uax/bidi"
)

func ExampleResolvedLevels_LogicalSegments() {
	text := "one two THREE FOUR FIVE"
	// TestMode makes uppercase letters stand for RTL text in this example.
	levels := bidi.ResolveParagraph(strings.NewReader(text), nil, bidi.TestMode(true))
	styles := []struct {
		from, to uint64
		font     string
	}{
		{0, 14, "regular"},
		{14, 18, "bold"},
		{18, uint64(len(text)), "regular"},
	}
	for segment := range levels.LogicalSegments() {
		for _, style := range styles {
			from := max(segment.From, style.from)
			to := min(segment.To, style.to)
			if from < to {
				// Shape text[from:to] using style.font and segment.Dir.
				fmt.Printf("%s %s [%d:%d] %q\n",
					segment.Dir, style.font, from, to, text[from:to])
			}
		}
	}
	// Use the measured items to choose line breaks, then Split and Reorder.

	// Output:
	// L2R regular [0:8] "one two "
	// R2L regular [8:14] "THREE "
	// R2L bold [14:18] "FOUR"
	// R2L regular [18:23] " FIVE"
}
