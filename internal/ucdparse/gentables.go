package ucdparse

import (
	"bytes"
	"fmt"
	"unicode"
)

// This is a very rough implementation.
// Creating Unicode tables is a rare task, and I do not plan to distribute
// this little CLI.

// We collect range-tables per character category.
// rangeTables will map category->table
var rangeTables map[string]*RangeTableCollector // one for each range table to generate

// ---------------------------------------------------------------------------

// RangeTableCollector is a type to collect character ranges during iteration of
// UCD files, and later output them to Go source code.
type RangeTableCollector struct {
	Cat              string // character category
	cnt, latinOffset int    // range count and unicode.RangeTable.LatinOffset
	switch32         int    // range item where to switch to int32 size
	ranges           [][2]rune
	lo, hi           rune // low and high bound of current range
}

// Append a range of runes to a range table collector. A single
// character is denoted by l == r.
//
func (rt *RangeTableCollector) Append(l, r rune) {
	if l == rt.hi+1 {
		rt.hi = r // range extends previous range
		return
	}
	// check for switch points in range list
	if rt.latinOffset == 0 && rt.hi > unicode.MaxLatin1 {
		rt.latinOffset = rt.cnt
	}
	if rt.switch32 == 0 && (rt.lo > (1<<16) || rt.hi > (1<<16)) {
		// switch to range32
		rt.switch32 = rt.cnt
	}
	// append current range lo…hi to ranges
	rt.ranges = append(rt.ranges, [2]rune{rt.lo, rt.hi})
	rt.cnt++
	rt.lo, rt.hi = l, r
}

// Output creates Go source code for a range table.
func (rt *RangeTableCollector) Output(buf *bytes.Buffer) {
	printRangePreamble(buf, rt)
	for i, r := range rt.ranges {
		if rt.switch32 > 0 && i == rt.switch32 {
			fmt.Fprintf(buf, "\t},\n\tR32: []unicode.Range32{\n")
		}
		fmt.Fprintf(buf, "\t\t{%#04x, %#04x, 1},\n", r[0], r[1])
	}
	printRangePostamble(buf, rt)
}

func printRangePreamble(buf *bytes.Buffer, t *RangeTableCollector) {
	fmt.Fprintf(buf, "var _%s = &unicode.RangeTable{ ", t.Cat)
	fmt.Fprintf(buf, "// %d entries", t.cnt)
	fmt.Fprintf(buf, `
	R16: []unicode.Range16{
`)
}

func printRangePostamble(buf *bytes.Buffer, t *RangeTableCollector) {
	if t.latinOffset > 0 {
		fmt.Fprintf(buf, "\t},\n\tLatinOffset: %d,\n}\n\n", t.latinOffset)
	} else {
		fmt.Fprintf(buf, "\t},\n}\n\n")
	}
}
