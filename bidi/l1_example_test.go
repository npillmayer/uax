package bidi_test

import (
	"fmt"
	"strings"

	"github.com/npillmayer/uax/bidi"
)

func ExampleResolvedLevels_ApplyL1() {
	text := "Go אבג  דה"
	levels := bidi.ResolveParagraph(strings.NewReader(text), nil)
	// The line breaker places the first line's end after the two spaces.
	lineEnd := uint64(len("Go אבג  "))
	resetFrom := uint64(len("Go אבג"))
	first, rest := levels.Clone().Split(lineEnd, false)
	if err := first.ApplyL1(resetFrom, lineEnd); err != nil {
		panic(err)
	}
	for _, run := range first.Reorder().Runs {
		for segment := range run.Segments(run.IsOpposite(bidi.LeftToRight)) {
			fmt.Printf("%s [%d:%d] %+q\n", segment.Dir, segment.From, segment.To,
				text[segment.From:segment.To])
		}
	}
	// This particular final line has no intervals requiring L1.
	_ = rest.Reorder()
	// Output:
	// L2R [0:3] "Go "
	// R2L [3:9] "\u05d0\u05d1\u05d2"
	// L2R [9:11] "  "
}
