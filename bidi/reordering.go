package bidi

import (
	"fmt"
	"iter"
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

// Clone returns an independent deep copy of the resolved levels, including
// their embedding direction and all nested isolates. Splitting or reordering
// either copy does not affect the other.
//
// Keep the original paragraph result and clone it once per layout attempt,
// before calling Split or Reorder. Clone copies the current state; it does not
// restore logical order if the receiver has already been reordered.
// It takes time and space proportional to the stored scraps and child slices.
// The source text is owned by the client and is not copied.
func (rl *ResolvedLevels) Clone() *ResolvedLevels {
	cloned := *rl
	cloned.scraps = cloneScraps(rl.scraps)
	return &cloned
}

func cloneScraps(scraps []scrap) []scrap {
	if scraps == nil {
		return nil
	}
	cloned := make([]scrap, len(scraps))
	copy(cloned, scraps)
	for i, s := range scraps {
		if s.children != nil {
			cloned[i].children = make([][]scrap, len(s.children))
			for j, child := range s.children {
				cloned[i].children[j] = cloneScraps(child)
			}
		}
	}
	return cloned
}

func (rl *ResolvedLevels) String() string {
	var b strings.Builder
	for _, s := range rl.scraps {
		b.WriteString(fmt.Sprintf("[%d-%s-%d] ", s.l, classString(s.bidiclz), s.r))
		//fmt.Fprintf(b, "[%d-%s-%d] ", s.l, classString(s.bidiclz), s.r)
	}
	return b.String()
}

// Level is a struct representing a resolved level run.
//
// See [ResolvedLevels.Range], an iterator over the resolved levels.
type Level struct {
	BidiClass bidi.Class
	From, To  int
}

// Range is an iterator over the resolved levels, yielding a [Level] struct
// for each resolved level run.
//
// This line (for an embedding context of LTR):
//
//	"Look out: זה כיף סתם לשמוע איך תנצח קרפד עץ טוב בגן"
//
// will produce the following levels:
//
//	[0-LRI-0] [0-L-10] [10-R-83] [83-PDI-83]
func (rl *ResolvedLevels) Range() iter.Seq2[int, Level] {
	return func(yield func(int, Level) bool) {
		for i, s := range rl.scraps {
			level := Level{
				BidiClass: s.bidiclz,
				From:      int(s.l),
				To:        int(s.r),
			}
			if !yield(i, level) {
				break
			}
		}
	}
}

// LogicalSegments returns text segments with resolved directions in source
// order, descending into nested isolates. Clients can intersect these byte
// intervals with font and style spans before shaping and choosing line breaks.
// European and Arabic number runs have direction LeftToRight.
//
// Isolate delimiters and zero-length intervals are omitted. In-text isolate
// controls therefore leave gaps in the returned byte intervals; out-of-line
// markup consumes no bytes. A paragraph containing only empty isolates yields
// no segments. Adjacent segments with the same direction are not
// necessarily merged, so isolate boundaries remain available to the client.
//
// Iterate before calling Reorder, which changes source order. After Split,
// iterate over its returned parts, using rebased coordinates if requested.
// LogicalSegments does not modify the backing data. Its iterator supports
// repeated and independent traversals and early exit, provided the resolved
// levels are not otherwise modified.
func (rl *ResolvedLevels) LogicalSegments() iter.Seq[Segment] {
	return func(yield func(Segment) bool) {
		walkLogicalSegments(rl.scraps, rl.embedding, yield)
	}
}

// walkLogicalSegments substitutes each child isolate for the corresponding
// part of its enclosing scrap, retaining the enclosing direction in the gaps.
// The scraps and their children must still be in logical order.
func walkLogicalSegments(scraps []scrap, embedding Direction, yield func(Segment) bool) bool {
	embedding = findEmbeddingDir(scraps, embedding)
	for _, s := range scraps {
		if isisolate(s) || s.bidiclz == cNULL {
			continue
		}
		dir := directionFromBidiClass(s, embedding)
		from := s.l
		for _, child := range s.children {
			if len(child) == 0 {
				continue
			}
			if from < child[0].l {
				if !yield(Segment{Dir: dir, From: uint64(from), To: uint64(child[0].l)}) {
					return false
				}
			}
			if !walkLogicalSegments(child, embedding, yield) {
				return false
			}
			from = child[len(child)-1].r
		}
		if from < s.r {
			if !yield(Segment{Dir: dir, From: uint64(from), To: uint64(s.r)}) {
				return false
			}
		}
	}
	return true
}

// Split divides resolved levels at a UTF-8 byte position. Text at the cut
// belongs to the suffix. Both results retain the paragraph's embedding
// direction, and nested isolates are partitioned recursively. Continuation
// delimiters are synthetic, zero-length markers at the new line boundaries.
//
// Empty input produces two empty results. A cut at or before the start produces
// an empty prefix; a cut at or after the end produces an empty suffix. The
// caller must choose a valid text boundary; Split does not validate UTF-8 or
// grapheme boundaries or find line-break opportunities.
//
// If shift0 is true, suffix positions (including nested isolates) are reduced
// by at, for use with a separately sliced text buffer. Otherwise both results
// retain their original paragraph offsets.
//
// Split consumes its receiver's backing data. Use Clone before splitting when
// the original levels must remain available. Each returned part can be split
// further or reordered independently.
func (rl *ResolvedLevels) Split(at uint64, shift0 bool) (*ResolvedLevels, *ResolvedLevels) {
	var prefix, suffix []scrap
	// Compare before narrowing to charpos so a large cut cannot wrap around.
	if len(rl.scraps) > 0 && at >= uint64(rl.scraps[len(rl.scraps)-1].r) {
		prefix = rl.scraps
	} else {
		prefix, suffix = split(rl.scraps, charpos(at))
	}
	if shift0 && len(suffix) > 0 {
		shiftzero(suffix, charpos(at))
	}
	return &ResolvedLevels{embedding: rl.embedding, scraps: prefix},
		&ResolvedLevels{embedding: rl.embedding, scraps: suffix}
}

func split(scraps []scrap, at charpos) ([]scrap, []scrap) {
	if len(scraps) == 0 {
		return nil, nil
	}
	if at <= scraps[0].l {
		return nil, scraps
	}
	if at >= scraps[len(scraps)-1].r {
		return scraps, nil
	}

	opening := scraps[0]
	hasOpening := opening.bidiclz == bidi.LRI || opening.bidiclz == bidi.RLI || opening.bidiclz == bidi.FSI
	prefix := scraps[:0] // reuse storage only after its source element is read
	suffix := make([]scrap, 0, len(scraps)+1)
	if hasOpening {
		// Retain the isolate direction, but never replay its original control
		// bytes on a later line. The original opening stays with the prefix.
		opening.l, opening.r = at, at
		opening.children = nil
		suffix = append(suffix, opening)
	}
	for _, s := range scraps {
		switch {
		case s.r <= at:
			prefix = append(prefix, s)
		case s.l >= at:
			suffix = append(suffix, s)
		default:
			left, right := s, s // preserve bidi class and directional context
			left.r, right.l = at, at
			left.children, right.children = nil, nil
			for _, child := range s.children {
				before, after := split(child, at)
				if len(before) > 0 {
					left.children = append(left.children, before)
				}
				if len(after) > 0 {
					right.children = append(right.children, after)
				}
			}
			prefix = append(prefix, left)
			suffix = append(suffix, right)
		}
	}
	if hasOpening {
		prefix = append(prefix, scrap{l: at, r: at, bidiclz: bidi.PDI})
	}
	return prefix, suffix
}

func shiftzero(scraps []scrap, offset charpos) {
	for i := range scraps {
		s := &scraps[i]
		s.l = validcharpos(s.l, offset)
		s.r = validcharpos(s.r, offset)
		s.context.matchPos = validcharpos(s.context.matchPos, offset)
		for _, child := range s.children {
			shiftzero(child, offset)
		}
	}
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
			tracer().Debugf("scrap has child = %v", ch)
			if dir != embedded {
				tracer().Debugf("child dir is different from embedding")
			}
			reorder(ch, 0, len(ch), dir)
			tracer().Debugf("reordered child = %v", ch)
			if dir != embedded {
				tracer().Debugf("reversing total child %v", ch)
				reverseWithoutIsolates(ch, 0, len(ch))
				tracer().Debugf("                child = %v", ch)
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
				tracer().Debugf("reverse(%d, %d)", startRunR, pos)
				reverse(scraps, startRunR, pos)
				startRunR = 0
			} else if pos == j {
				tracer().Debugf("EOF reverse(%d, %d)", startRunR, pos+1)
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
	tracer().Debugf("reverse w/ isolates %d…%d : %v", i, j, scraps)
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
	tracer().Debugf("will flatten( %v ), emb=%v", scraps, embedding)
	var runs []Run
	var last Run
	if len(scraps) > 0 && isisolate(scraps[0]) {
		embedding = directionFromBidiClass(scraps[0], embedding)
	}
	for _, s := range scraps {
		tracer().Debugf("s=%v, |ch|=%d, last=%v", s, len(s.children), last)
		if len(s.children) == 0 {
			last, runs = extendRuns(s, last, runs, embedding)
			tracer().Debugf("runs = %v", runs)
			continue
		}
		var cut scrap
		for _, ch := range s.children {
			if len(ch) == 0 {
				continue
			}
			tracer().Debugf("scrap has child = %v ------------------", ch)
			cut, s = cutScrapAt(s, ch)
			last, runs = extendRuns(cut, last, runs, embedding)
			//chrun := flatten(ch, embedding) // TODO embedding -> last.Dir ?
			chruns := flatten(ch, last.Dir) // TODO embedding -> last.Dir ?
			tracer().Debugf("flattened child = %v", chruns)
			tracer().Debugf("----------------------------------------------------------------")
			last, runs = appendRuns(chruns, last, runs)
			tracer().Debugf("runs+child=%v, last=%v", runs, last)
		}
		tracer().Debugf("after children: s=%v, runs=%v", s, runs)
		last, runs = extendRuns(s, last, runs, embedding)
		tracer().Debugf("runs = %v", runs)
	}
	return runs
}

func extendRuns(s scrap, last Run, runs []Run, embedding Direction) (Run, []Run) {
	tracer().Debugf("extendRuns(%v), last=%v, runs=%v", s, last, runs)
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
		tracer().Debugf("extending %v with %v", runs, last)
		runs[len(runs)-1] = last
	} else { // have to switch directions
		tracer().Debugf("switching dir")
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
	if len(runs) == 0 {
		// An isolate can precede all outer text, including when its only
		// contents are delimiters. There is no preceding run to concatenate.
		return childRuns[len(childRuns)-1], childRuns
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
	tracer().Debugf("=====reorder done, flatten ========")
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

// Segment describes a text interval in the original, unshuffled input.
// From and To are UTF-8 byte offsets; From is inclusive and To is exclusive.
// Dir is the direction in which the segment should be rendered.
type Segment struct {
	Dir      Direction
	From, To uint64
}

// Segments returns an iterator over the text segments contained in a run.
// If reverse is true, segments are visited in reverse order. The characters
// within each segment remain in their original logical order.
//
// For left-to-right output, a client can iterate with:
//
//	for segment := range run.Segments(run.IsOpposite(bidi.LeftToRight)) {
//	    text := paragraph[segment.From:segment.To]
//	    // Shape and draw text using segment.Dir.
//	}
//
// Iteration does not modify the run. The returned iterator can be reused and
// supports independent traversals, provided the run is not otherwise modified.
func (r *Run) Segments(reverse bool) iter.Seq[Segment] {
	return func(yield func(Segment) bool) {
		for i := 0; i < len(r.scraps); i++ {
			index := i
			if reverse {
				index = len(r.scraps) - 1 - i
			}
			s := r.scraps[index]
			if !yield(Segment{Dir: r.Dir, From: uint64(s.l), To: uint64(s.r)}) {
				return
			}
		}
	}
}

// SegmentIterator iterates over the text segments contained in a run.
// Runs are the product of a re-ordering of text, which may lead to segments of text
// to be shuffled around. A segment starts and ends at text positions of the unshuffled
// text. Clients will need this information to create the correct visual order
// of text segments.
// For range-based iteration without modifying the run, use [Run.Segments].
type SegmentIterator struct {
	//run      *Run
	Dir      Direction
	scraps   []scrap
	reverse  bool
	interval int
	eof      bool
}

// SegmentIterator creates an interator for the text segments contained within a
// Bidi run. Assume the output devices current direction is set to left-to-right:
//
//	it := run.SegmentIterator(run.IsOpposite(bidi.LeftToRight))
//	for it.Next() {
//	    dir, from, to := it.Segment()
//	    var segment string
//	    segment = myGetSegString(from, to)  // client func to get the text-segment by positions
//	    if dir == bidi.RightToLeft {
//	        segment = reverse(segment)      // client func to reverse graphemes of a string
//	    }
//	    …                                   // visual output of segment
//	}
//
// Clients of this package should proceed like this for every Run of an Ordering.
//
// Parameter reverse should be set to true if the clients expects fragments of
// the run in reverse order. This will be the case in situations where the direction
// of the run is opposite to the visual direction of an output device and the
// application has to handle visual re-ordering.
//
// This method is retained for compatibility. Unlike [Run.Segments], passing
// true reverses the run's backing segment slice in place.
func (r *Run) SegmentIterator(reverse bool) *SegmentIterator {
	it := &SegmentIterator{
		//run:     r,
		Dir:     r.Dir,
		scraps:  r.scraps,
		reverse: reverse,
	}
	if reverse {
		mid := len(it.scraps) / 2
		for i := 0; i < mid; i++ {
			j := len(it.scraps) - i - 1
			it.scraps[i], it.scraps[j] = it.scraps[j], it.scraps[i]
		}
	}
	return it
}

// IsOpposite returns true if the run's direction is oppostite to the
// given direction.
func (r *Run) IsOpposite(dir Direction) bool {
	return dir != r.Dir
}

// Next proceeds the iterator to the next segment of text.
func (it *SegmentIterator) Next() bool {
	if it.interval >= len(it.scraps) {
		return false
	}
	it.interval++
	if it.interval == len(it.scraps) {
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
		return it.Dir, 0, 0
	}
	s := it.scraps[it.interval-1]
	return it.Dir, uint64(s.l), uint64(s.r)
}

// ---------------------------------------------------------------------------

func validcharpos(p, diff charpos) charpos {
	if p > diff {
		return p - diff
	}
	return 0
}
