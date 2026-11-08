package bidi

import (
	"fmt"

	"golang.org/x/text/unicode/bidi"
)

// ApplyL1 resets [from, to) to the paragraph embedding level under UAX#9 rule L1.
// Positions are UTF-8 byte offsets in the receiver's coordinate system, including
// any rebasing performed by Split. Apply it to each line after Split and before
// Reorder, including the final line (or a paragraph occupying a single line).
// Multiple calls can reset disjoint intervals, such as whitespace and separators
// inside a line as well as trailing whitespace. Repeated and overlapping resets
// have the same effect as resetting their union.
//
// The caller identifies eligible intervals using the original bidi classes:
// segment and paragraph separators, and sequences of whitespace and/or isolate
// formatting characters preceding those separators or the end of the line.
// The package cannot check eligibility or character boundaries without the text.
// A reset inside an isolate uses the paragraph level, not the isolate's level.
// Isolate controls remain omitted from LogicalSegments.
//
// Reversed or out-of-range intervals return an error without changing the
// receiver. An empty interval within the receiver's bounds is a no-op; empty
// levels without stored positions accept only [0, 0).
// ApplyL1 modifies backing data. Use Clone before splitting and resetting when
// the original paragraph must remain available for another layout attempt.
func (rl *ResolvedLevels) ApplyL1(from, to uint64) error {
	var start, end uint64
	if len(rl.scraps) > 0 {
		start = uint64(rl.scraps[0].l)
		end = uint64(rl.scraps[len(rl.scraps)-1].r)
	}
	// Validate before narrowing to charpos or modifying any backing slices.
	if from > to || from < start || to > end {
		return fmt.Errorf("bidi: L1 interval [%d,%d) outside [%d,%d) or reversed", from, to, start, end)
	}
	if from == to {
		return nil
	}

	before, rest := split(rl.scraps, charpos(from))
	middle, after := split(rest, charpos(to))

	// Split preserves the nesting on either side, closing/reopening isolates
	// with synthetic markers. Remove only the outer paragraph wrappers when
	// joining the pieces; nested continuation markers must remain intact.
	before = paragraphContents(before)
	after = paragraphContents(after)
	opening := bidi.LRI
	if rl.embedding == RightToLeft {
		opening = bidi.RLI
	}
	joined := make([]scrap, 0, len(before)+len(middle)+len(after)+2)
	joined = append(joined, scrap{bidiclz: opening, l: charpos(start), r: charpos(start)})
	joined = append(joined, before...)
	joined = appendL1Scraps(joined, middle, rl.embedding)
	joined = append(joined, after...)
	joined = append(joined, scrap{bidiclz: bidi.PDI, l: charpos(end), r: charpos(end)})
	rl.scraps = joined
	return nil
}

// paragraphContents removes the synthetic paragraph delimiters produced by
// resolution and Split. Actual source controls, including empty isolates, remain.
func paragraphContents(scraps []scrap) []scrap {
	if len(scraps) > 0 {
		first := scraps[0]
		if first.l == first.r && (first.bidiclz == bidi.LRI || first.bidiclz == bidi.RLI) {
			scraps = scraps[1:]
		}
	}
	if len(scraps) > 0 {
		last := scraps[len(scraps)-1]
		if last.l == last.r && last.bidiclz == bidi.PDI {
			scraps = scraps[:len(scraps)-1]
		}
	}
	return scraps
}

// appendL1Scraps replaces nested intervals by their leaves in source order.
// Reset text must participate in reordering at the root, otherwise an enclosing
// isolate would still reverse it. Control identity is kept separate from its
// new direction, so resetting does not turn controls into shaping input.
func appendL1Scraps(dst, scraps []scrap, paragraph Direction) []scrap {
	base := bidi.L
	if paragraph == RightToLeft {
		base = bidi.R
	}
	appendLeaf := func(s scrap, from, to charpos) {
		if from >= to || s.bidiclz == cNULL {
			return
		}
		s.l, s.r = from, to
		s.children = nil
		s.context = dirContext{}
		s.l1Reset = isisolate(s)
		if !s.l1Reset {
			s.bidiclz = base
		}
		dst = append(dst, s)
	}
	for _, s := range scraps {
		from := s.l
		for _, child := range s.children {
			if len(child) == 0 {
				continue
			}
			appendLeaf(s, from, child[0].l)
			dst = appendL1Scraps(dst, child, paragraph)
			from = child[len(child)-1].r
		}
		appendLeaf(s, from, s.r)
	}
	return dst
}
