package bidi

import (
	"fmt"
	"strings"

	"golang.org/x/text/unicode/bidi"
)

// --- ResolvedLevels --------------------------------------------------------

// Resolving embedding levels consists of application of L1 to L4.
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
	embedding Direction // TODO set this from parser
	scraps    []scrap
}

func (rl *ResolvedLevels) String() string {
	var b strings.Builder
	for _, s := range rl.scraps {
		b.WriteString(fmt.Sprintf("[%d-%s-%d] ", s.l, classString(s.bidiclz), s.r))
	}
	return b.String()
}

// Split cuts a resolved level run into 2 pieces at position at. The character
// at position at will be the first character of the second (cut-off) piece.
//
// Clients typically use this for line-wrapping. Cut-off level runs (= lines) can then
// be reordered one by one.
//
// If parameter shift0 is set, all indices within resolved levels will be lowered by `at`,
// resulting in the first level to have a left boundary of zero. This is useful for
// cases where the clients splits the underlying text congruently to Bidi levels
// and characters are therefore “re-positioned”.
//
func (rl *ResolvedLevels) Split(at uint64, shift0 bool) (*ResolvedLevels, *ResolvedLevels) {
	prefix, suffix := split(rl.scraps, charpos(at))
	if shift0 {
		//suffix = shiftzero(suffix, charpos(at))
		shiftzero(&suffix, charpos(at))
		T().Debugf("resolved levels: shifted suffix levels = %v", suffix)
	}
	return &ResolvedLevels{scraps: prefix}, &ResolvedLevels{scraps: suffix}
}

func split(scraps []scrap, at charpos) ([]scrap, []scrap) {
	// resulting level runs end up unbalanced, i.e. the nested IRS are
	// split, too. In fact, we need to introduce a LRI/RLI at the beginning of
	// the right rest and can spare PDIs in left parts.
	T().Debugf("split @%d, irs = %v", at, scraps)
	if !irsContains(scraps, at) {
		return scraps, []scrap{}
	}
	for i, s := range scraps {
		if !s.contains(at) {
			continue
		}
		var restch [][]scrap
		if len(s.children) > 0 { // TODO omit this
			T().Debugf("split in %v with |ch|=%d", s, len(s.children))
			for j, ch := range s.children {
				if irsContains(ch, at) {
					prefix, suffix := split(ch, at)
					restch = [][]scrap{suffix}
					if j < len(s.children)-1 {
						restch = append(restch, s.children[j+1:]...)
					}
					s.children = s.children[:j]
					s.children = append(s.children, prefix)
					break
				}
			}
		}
		// children are split at this point
		// restch is set to right part of children split
		rest := scrap{r: s.r, bidiclz: s.bidiclz}
		s.r, rest.l = at, at
		rest.children = restch
		rseg := make([]scrap, len(scraps)-i+1)
		if isisolate(scraps[0]) {
			rseg[0] = scraps[0] // copy IRS start
			copy(rseg[1:], scraps[i:])
			scraps[i], rseg[1] = s, rest
		} else {
			copy(rseg, scraps[i:])
			rseg = rseg[:len(rseg)-1]
			scraps[i], rseg[0] = s, rest
		}
		prefix := scraps[:i+1]
		T().Debugf("prefix=%v", prefix)
		T().Debugf("suffix=%v", rseg)
		return scraps[:i+1], rseg
	}
	panic("split iterated over all scraps, did not find cut line")
}

func irsContains(scraps []scrap, pos charpos) bool {
	return scraps[0].l <= pos && scraps[len(scraps)-1].r > pos
}

func shiftzero(scraps *[]scrap, offset charpos) { //[]scrap {
	for i, s := range *scraps {
		s.l = validcharpos(s.l, offset)
		s.r = validcharpos(s.r, offset)
		(*scraps)[i] = s
		//T().Errorf("scrap = %v", s)
	}
	//T().Errorf("shifted = %v", scraps)
	//return scraps
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
	Dir    Direction // either LeftToRight or RightToLeft
	Length int64     // length of run in bytes
	scraps []scrap   //
	//L, R uint64    // left text position and right text position
}

func run(s scrap, embedding Direction) Run {
	return Run{
		Dir:    directionFromBidiClass(s, embedding),
		Length: int64(s.len()),
		scraps: []scrap{s},
	}
}

func (r Run) String() string {
	s := fmt.Sprintf("(%s %d", r.Dir, r.Length)
	for _, sc := range r.scraps {
		s += fmt.Sprintf(" %d…%d|%s", sc.l, sc.r, classString(sc.bidiclz))
	}
	s += ")"
	return s
}

func (r *Run) concat(other Run) {
	if other.Length == 0 {
		return
	}
	r.Length += other.Length
	if other.Dir != r.Dir {
		panic("cannot concat Run with Run of other direction")
	}
	lscr := r.scraps[len(r.scraps)-1]
	ofst := other.scraps[0]
	if lscr.r == ofst.l {
		lscr.r = ofst.r
		r.scraps[len(r.scraps)-1] = lscr
		r.scraps = append(r.scraps, other.scraps[1:]...)
	} else {
		r.scraps = append(r.scraps, other.scraps...)
	}
}

// reorder IRS destructively, recursively.
//
// Not sure i…j is really necessary
//
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
		//T().Debugf("scrap=%v, pos=%d, state=%d", s, pos, state)
		for _, ch := range s.children {
			dir := findEmbeddingDir(ch, embedded)
			T().Debugf("scrap has child = %v", ch)
			if dir != embedded {
				T().Debugf("child dir is different from embedding")
			}
			reorder(ch, 0, len(ch), dir)
			T().Debugf("reordered child = %v", ch)
			if dir != embedded {
				T().Debugf("reversing total child %v", ch)
				reverseWithoutIsolates(ch, 0, len(ch))
				T().Debugf("                child = %v", ch)
			}
		}
		switch state {
		case 0: // skipping e's, looking for o, EN, AN
			if level(s, embedded) > 0 {
				state = 1
				startRunR = pos
			}
		case 1: // collecting o, EN, AN
			if level(s, embedded) == 0 {
				state = 0
				T().Debugf("reverse(%d, %d)", startRunR, pos)
				reverse(scraps, startRunR, pos)
				startRunR = 0
			} else if pos == j {
				T().Debugf("EOF reverse(%d, %d)", startRunR, pos+1)
				reverse(scraps, startRunR, pos+1)
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
	for ; i < j; i, j = i+1, j-1 {
		scraps[i], scraps[j] = scraps[j], scraps[i]
	}
	return scraps
}

func reverseWithoutIsolates(scraps []scrap, i, j int) []scrap {
	if len(scraps) == 0 {
		return scraps
	}
	if i > j {
		i, j = j, i
	}
	j = min(j, len(scraps))
	if isisolate(scraps[i]) {
		i++
	}
	if isisolate((scraps[j-1])) {
		j--
	}
	T().Debugf("reverse w/ isolates %d…%d : %v", i, j, scraps)
	return reverse(scraps, i, j)
}

func findEmbeddingDir(scraps []scrap, inherited Direction) Direction {
	if len(scraps) == 0 {
		return inherited
	}
	if isisolate(scraps[0]) {
		switch scraps[0].bidiclz {
		case bidi.LRI:
			return LeftToRight
		case bidi.RLI:
			return RightToLeft
		case bidi.PDI:
			panic("PDI is first scrap in child run, strange")
		case bidi.FSI:
			panic("bidi.FSI not yet supported")
		}
	}
	return inherited
}

func flatten(scraps []scrap, embedding Direction) []Run {
	T().Debugf("will flatten( %v ), emb=%v", scraps, embedding)
	var runs []Run
	var last Run
	if len(scraps) > 0 && isisolate(scraps[0]) {
		embedding = directionFromBidiClass(scraps[0], embedding)
	}
	for _, s := range scraps {
		T().Debugf("s=%v, |ch|=%d, last=%v", s, len(s.children), last)
		if len(s.children) == 0 {
			last, runs = extendRuns(s, last, runs, embedding)
			T().Debugf("runs = %v", runs)
			continue
		}
		var cut scrap
		for _, ch := range s.children {
			if len(ch) == 0 {
				continue
			}
			T().Debugf("scrap has child = %v ------------------", ch)
			cut, s = cutScrapAt(s, ch)
			last, runs = extendRuns(cut, last, runs, embedding)
			//chrun := flatten(ch, embedding) // TODO embedding -> last.Dir ?
			chruns := flatten(ch, last.Dir) // TODO embedding -> last.Dir ?
			T().Debugf("flattened child = %v", chruns)
			T().Debugf("----------------------------------------------------------------")
			last, runs = appendRuns(chruns, last, runs)
			T().Debugf("runs+child=%v, last=%v", runs, last)
		}
		T().Debugf("after children: s=%v, runs=%v", s, runs)
		last, runs = extendRuns(s, last, runs, embedding)
		T().Debugf("runs = %v", runs)
	}
	return runs
}

func extendRuns(s scrap, last Run, runs []Run, embedding Direction) (Run, []Run) {
	T().Debugf("extendRuns(%v), last=%v, runs=%v", s, last, runs)
	if s.len() == 0 { // important. must correlate to semantics of runappend()
		return last, runs
	}
	dir := directionFromBidiClass(s, embedding)
	if last.Length == 0 { // we are at the start of a sequence
		// last.L = uint64(s.l)
		// last.R = uint64(s.r)
		last = run(s, embedding)
		//last.concat(last)
		//last.Length = int64(s.len())
		runs = runappend(runs, last)
	} else if last.Dir == dir { // just extend last with s
		//last.R = uint64(s.r)
		last.concat(run(s, embedding))
		T().Debugf("extending %v with %v", runs, last)
		runs[len(runs)-1] = last
	} else { // have to switch directions
		T().Debugf("switching dir")
		last = run(s, embedding)
		// last = Run{
		// 	// L:   uint64(s.l),
		// 	// R:   uint64(s.r),
		// 	Dir:    dir,
		// 	Length: int64(s.len()),
		// }
		runs = runappend(runs, last)
	}
	return last, runs
}

func cutScrapAt(s scrap, ch []scrap) (scrap, scrap) {
	if len(ch) == 0 {
		panic("internal inconsistency: ch may not be empty")
	}
	cut := scrap{
		l:       s.l,
		r:       ch[0].l,
		bidiclz: s.bidiclz,
	}
	rest := scrap{
		l:       ch[len(ch)-1].r,
		r:       s.r,
		bidiclz: s.bidiclz,
	}
	return cut, rest
}

func appendRuns(childRuns []Run, last Run, runs []Run) (Run, []Run) {
	if len(childRuns) == 0 {
		return last, runs
	}
	if childRuns[0].Dir == last.Dir {
		//last.Length = childRuns[0].Length
		last.concat(childRuns[0])
		runs = append(runs, childRuns[1:]...)
	} else {
		runs = append(runs, childRuns...)
	}
	last = runs[len(runs)-1]
	return last, runs
}

func runappend(runs []Run, r Run) []Run {
	if r.Length > 0 {
		runs = append(runs, r)
	}
	return runs
}

func directionFromBidiClass(s scrap, embedding Direction) Direction {
	switch s.bidiclz {
	case bidi.L, bidi.LRI:
		return LeftToRight
	case bidi.R, bidi.RLI:
		return RightToLeft
	case bidi.EN, bidi.AN:
		return LeftToRight
	case bidi.PDI:
		return embedding
	}
	return embedding
}

// An Ordering holds the computed visual order of bidi-runs of a paragraph of text.
type Ordering struct {
	Runs []Run
}

// Reorder reorders runs of resolved levels and returns an ordering reflecting runs
// of characters with either L2R or R2L direction.
func (rl *ResolvedLevels) Reorder() *Ordering {
	if len(rl.scraps) == 0 {
		return &Ordering{}
	}
	rscr := reorder(rl.scraps, 0, len(rl.scraps), rl.embedding)
	T().Debugf("=====reorder done, flatten ========")
	r := flatten(rscr, rl.embedding)
	return &Ordering{Runs: r}
}

// DirectionAt returns the text direction at byte position pos.
func (rl *ResolvedLevels) DirectionAt(pos uint64) Direction {
	for _, s := range rl.scraps {
		if uint64(s.l) <= pos && pos < uint64(s.r) {
			return directionFromBidiClass(s, LeftToRight)
		}
	}
	return LeftToRight
}

// SegmentIterator iterates over the text segments contained in a run.
// Runs are the product of a re-ordering of text, which may lead to segments of text
// to be shuffled around. A segment starts and ends at text positions of the unshuffled
// text. Clients will need this information to create the correct visual order
// of text segments.
type SegmentIterator struct {
	run      *Run
	interval int
	eof      bool
}

// SegmentIterator creates an interator for the text segments contained within a
// Bidi run.
//
//     it := run.SegmentIterator()
//     for it.Next() {
//         dir, from, to := it.Segment()
//         var segment string
//         segment = myGetSegString(from, to)  // client func to get the text-segment by positions
//         if dir == bidi.LeftToRight {
//             segment = reverse(segment)
//         }
//         …                                   // visual output of segment
//     }
//
// Clients of this package should proceed like this for every Run of an Ordering.
//
func (r *Run) SegmentIterator() *SegmentIterator {
	return &SegmentIterator{
		run: r,
	}
}

// Next proceeds the iterator to the next segment of text.
func (it *SegmentIterator) Next() bool {
	if it.interval >= len(it.run.scraps) {
		return false
	}
	it.interval++
	if it.interval == len(it.run.scraps) {
		it.eof = true
	}
	return true
}

// EOF returns true if the iterator has read the last segment.
func (it *SegmentIterator) EOF() bool {
	return it.eof
}

// Segment returns the bounds of the current segment of text.
func (it *SegmentIterator) Segment() (Direction, uint64, uint64) {
	if it.interval == 0 {
		return it.run.Dir, 0, 0
	}
	s := it.run.scraps[it.interval-1]
	return it.run.Dir, uint64(s.l), uint64(s.r)
}

// ---------------------------------------------------------------------------

func validcharpos(p, diff charpos) charpos {
	if p > diff {
		return p - diff
	}
	return 0
}
