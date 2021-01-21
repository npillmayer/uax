package bidi

import (
	"fmt"
	"strings"

	"golang.org/x/text/unicode/bidi"
)

// --- ResolvedLevels --------------------------------------------------------

//
// We will disregard rule L1 for now and leave it to the client. It would be
// possible to remember whitespace at IRS boundaries, but we'd rather avoid
// remembering WS runs in the middle of the paragraph, as required by rule L1,
// because any WS may end up next to a boundary after line wrap.
//
// We support reversing scraps and splitting up sequences of scraps, where
// splits may occur mid-scrap. This is better done by cords, but we'll have
// to provide an interface for clients not using cords as well. However,
// we have to support an appropriate cord leaf type, but coupling to package
// cord should not happen here. What kind of interface can we provide to
// enable package cord to define a leaf type?

// ResolvedLevels is a type for holding the result of phase 3.3
// “Resolving Embedded Levels”.
type ResolvedLevels struct {
	scraps []scrap
}

func (rl *ResolvedLevels) String() string {
	var b strings.Builder
	for _, s := range rl.scraps {
		b.WriteString(fmt.Sprintf("[%d-%s-%d] ", s.l, classString(s.bidiclz), s.r))
	}
	return b.String()
}

func (rl *ResolvedLevels) Split(uint64) (*ResolvedLevels, *ResolvedLevels) {
	// resulting level runs end up unbalanced, i.e. the nested IRS are
	// split, too.
	//
	// Is ordering a separate step or do we return 2 orderings directly?
	// usually only one fragment will be finished, the other one will be
	// split further, thus ordering should probably a separate step.
	return nil, nil
}

// The Reordering Phase of the UAX#9 algorithm is basically building a tree
// hierarchy, where nesting depth is reflected by embedding level numbers.
// In our implementation we already have a real tree structure for nested
// isolating run sequences in place and just need to handle L, R, EN and AN
// runs and their nestings.
//
// Table 5. Resolving Implicit Levels (see section 3.3.6)
//
// Type  | Embedding Level
// ------+-----------------
//       |   Even    Odd
// L     |   EL      EL+1
// R     |   EL+1    EL
// AN    |   EL+2    EL+1
// EN    |   EL+2    EL+1
//
// According to this table, handling L2R and R2l is not symmetric. Therefore we will
// set up a case switch depending on the embedding level of a IRS.
// Case L2R:
//    It remains to treat R-runs as having nesting level 1 and number runs
//    as having nesting level 2.
//    run=( (EN|AN)+ |L)+  ⇒  rev(run where rev(numbers))
// Case R2L:
//    We simply reverse runs of Ls and numbers: run=(EN|AN|L)+  ⇒  rev(run)
//

// A Run represents a directional run of text
// (i.e., a continuous sequence of characters of a single direction).
// Type Run holds the positions of characters, not the characters themselves.
type Run struct {
	Dir  Direction // either LeftToRight or RightToLeft
	L, R uint64    // left text position and right text position
}

// reorder IRS destructively
func reorder(scraps []scrap, i, j int, embedded Direction) []scrap {
	if len(scraps) <= i {
		return scraps
	} else if i > j {
		i, j = j, i
	}
	j = min(j-1, len(scraps)-1)
	pos, startRunR := 0, 0
	state := 0       // state of a super-simple finite automaton
	for state != 2 { // state 2 = EOF
		s := scraps[pos]
		switch state {
		case 0: // skipping e's, looking for o, EN, AN
			if level(s, embedded) > 0 {
				state = 1
				startRunR = pos
			}
		case 1: // collecting o, EN, AN
			if pos == j || level(s, embedded) == 0 {
				state = 0
				reverse(scraps, startRunR, pos+1)
				startRunR = 0
			}
		}
		if pos == j || s.bidiclz == cNULL {
			state = 2
		}
		pos++
	}
	return scraps
}

func level(s scrap, embedded Direction) int {
	if s.bidiclz == bidi.EN || s.bidiclz == bidi.AN {
		return 2
	}
	if (s.bidiclz == bidi.L && embedded == RightToLeft) || (s.bidiclz == bidi.R && embedded == LeftToRight) {
		return 1
	}
	return 0
}

// reverse ordering of [i,j)
func reverse(scraps []scrap, i, j int) []scrap {
	if i > j {
		i, j = j, i
	}
	j = min(j-1, len(scraps)-1)
	//mid := (j - i) / 2
	//T().Debugf("i=%d, j=%d, mid=%d", i, j, mid)
	//for ; i <= mid && i < j; i, j = i+1, j-1 {
	for ; i < j; i, j = i+1, j-1 {
		scraps[i], scraps[j] = scraps[j], scraps[i]
	}
	return scraps
}

// An Ordering holds the computed visual order of bidi-runs of a paragraph of text.
type Ordering struct {
	runs []Run
}

// Reorder reorders runs of resolved levels and returns an ordering reflecting runs
// of characters with either L2R or R2L direction.
func (rl *ResolvedLevels) Reorder() *Ordering {
	//
	return nil
}
