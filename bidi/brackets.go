package bidi

import (
	"fmt"
	"sync"

	"github.com/npillmayer/uax/internal/tracing"
)

// BD16MaxNesting is the maximum stack depth for rule BS16 as defined in UAX#9.
const BD16MaxNesting = 63

func isbracket(s scrap) bool {
	clz := s.bidiclz
	return clz == cBRACKO || clz == cBRACKC
}

// --- Brackets and bracket stack --------------------------------------------

// Brackets require a disproportionate amount of work in UAX#9. It reads:
//
// A bracket pair is a pair of characters consisting of an opening paired bracket
// and a closing paired bracket such that the Bidi_Paired_Bracket property value
// of the former or its canonical equivalent equals the latter or its canonical
// equivalent and which are algorithmically identified at specific text positions
// within an isolating run sequence.
//
// The following algorithm identifies all of the bracket pairs in a given isolating run sequence:
//
// * Create a fixed-size stack for exactly 63 elements each consisting of a bracket
//   character and a text position. Initialize it to empty.
// * Create a list for elements each consisting of two text positions, one for an opening
//   paired bracket and the other for a corresponding closing paired bracket. Initialize
//   it to empty.
// * Inspect each character in the isolating run sequence in logical order.
//   - If an opening paired bracket is found and there is room in the stack, push its
//     Bidi_Paired_Bracket property value and its text position onto the stack.
//   - If an opening paired bracket is found and there is no room in the stack, stop
//     processing BD16 for the remainder of the isolating run sequence.
//   - If a closing paired bracket is found, do the following:
// 	   1. Declare a variable that holds a reference to the current stack element and
//        initialize it with the top element of the stack.
// 	   2. Compare the closing paired bracket being inspected or its canonical equivalent
//        to the bracket in the current stack element.
// 	   3. If the values match, meaning the two characters form a bracket pair, then
// 	      . Append the text position in the current stack element together with the
//          text position of the closing paired bracket to the list.
// 	      . Pop the stack through the current stack element inclusively.
// 	   4. Else, if the current stack element is not at the bottom of the stack, advance
//        it to the next element deeper in the stack and go back to step 2.
// 	   5. Else, continue with inspecting the next character without popping the stack.
// * Sort the list of pairs of text positions in ascending order based on the text position of the opening paired bracket.
//
// Examples of bracket pairs:
//
// 	Text                Pairings
// 	1 2 3 4 5 6 7 8
// 	a ) b ( c           None
// 	a ( b ] c           None
// 	a ( b ) c           2-4
// 	a ( b [ c ) d ]     2-6
// 	a ( b ] c ) d       2-6
// 	a ( b ) c ) d       2-4
// 	a ( b ( c ) d       4-6
// 	a ( b ( c ) d )     2-8, 4-6
// 	a ( b { c } d )     2-8, 4-6

// We use a special class to handle discovery of bracket pairs.
type bracketPairHandler struct {
	stack    bracketStack        // stack to trace bracket nesting
	pairings pairingsList        // result list of matching bracket pairings
	mx       sync.Mutex          // guards pairings, as parser will access them, too
	next     *bracketPairHandler // handlers will be stacked for nested isolate runs
	firstpos charpos             // text position of first character in isolating run sequence
	lastpos  charpos             // text position of PDI
}

// This is the stack to perform the algorithm described above
type bracketStack []brktpos
type brktpos struct {
	//pos  charpos // position of an opening bracket
	opening scrap // opening bracket represented as a scrap
	pair    bracketPair
}

// This is the list of pairings found
type pairingsList []pairing
type pairing struct {
	brktpos
	closing scrap // closing bracket represented as a scrap
}

func makeBracketPairHandler(first charpos, previous *bracketPairHandler) *bracketPairHandler {
	h := &bracketPairHandler{
		stack:    make(bracketStack, 0, BD16MaxNesting),
		pairings: make(pairingsList, 0, 8),
		firstpos: first,
	}
	if previous != nil {
		previous.next = h
	}
	return h
}

func (pr pairing) String() string {
	return fmt.Sprintf("[%#U,%#U] at %d → %s", pr.pair.o, pr.pair.c, pr.opening.l, pr.closing)
}

// pushOpening pushes an opening bracket and its position onto the stack.
func (bph *bracketPairHandler) pushOpening(r rune, s scrap) bool {
	var ok bool
	ok, bph.stack = bph.stack.push(r, s)
	return ok
}

func (bph *bracketPairHandler) findPair(r rune, s scrap) (found bool, pos charpos) {
	var b brktpos
	if found, b, bph.stack = bph.stack.popWith(r, s.l); !found {
		return
	}
	newPairing := pairing{
		brktpos: b,
		closing: s,
	}
	inx := 0
	bph.mx.Lock()
	defer bph.mx.Unlock()
	for i, pair := range bph.pairings {
		if newPairing.opening.l < pair.opening.l {
			inx = i
		}
	}
	bph.pairings = append(bph.pairings[:inx], append([]pairing{newPairing}, bph.pairings[inx:]...)...)
	return
}

func (bs bracketStack) push(r rune, s scrap) (bool, bracketStack) {
	// TODO BS16Max is the limit per isolating run sequence, not overall
	if len(bs) >= BD16MaxNesting { // skip in case of stack overflow, as defined in UAX#9
		return false, bs
	}
	if s.bidiclz != cBRACKO {
		return false, bs
	}
	// TODO put bracket list in sutable data structure (map like) ?
	for _, pair := range uax9BracketPairs { // double check for UAX#9 brackets
		if pair.o == r {
			b := brktpos{opening: s, pair: pair}
			return true, append(bs, b)
		}
	}
	tracing.Errorf("Push of %v failed, not found as opening bracket", r)
	return false, bs
}

// popWith checks of an opening bracket on the bracket stack matching a given
// closing bracket. It performs steps 1–5 from the algorithm described above.
func (bs bracketStack) popWith(b rune, pos charpos) (bool, brktpos, bracketStack) {
	tracing.Debugf("popWith: rune=%v, bracket stack is %v", b, bs)
	if len(bs) == 0 {
		return false, brktpos{}, bs
	}
	i := len(bs) - 1
	for i >= 0 { // start at TOS, possible skip unclosed opening brackets
		if bs[i].pair.c == b {
			open := bs[i] //.pos
			bs = bs[:i]
			return true, open, bs
		}
		i--
	}
	return false, brktpos{}, bs
}

//func (bph *bracketPairHandler) FindBracketPairing(s scrap, open bool) (pairing, bool) {
func (bph *bracketPairHandler) FindBracketPairing(s scrap) (pairing, bool) {
	bph.mx.Lock()
	defer bph.mx.Unlock()
	for _, pair := range bph.pairings {
		if s.bidiclz == cBRACKO { // scrap is opening bracket
			if pair.opening.l == s.l {
				return pair, true
			}
		} else { // scrap is closing bracket
			if pair.closing.l == s.l {
				return pair, true
			}
		}
	}
	return pairing{}, false
}

// Rule N0 may change the bidi class of bracket scraps to a strong type.
// The parser rule for N0 will call changeOpeningBracketClass in such
// cases.
func (bph *bracketPairHandler) changeOpeningBracketClass(s scrap) {
	bph.mx.Lock()
	defer bph.mx.Unlock()
	for i, pair := range bph.pairings {
		if pair.opening.l == s.l {
			bph.pairings[i].opening.bidiclz = s.bidiclz
		}
	}
}

// UAX#9 rule W7 may change scraps of class EN to class L. However, the
// matching of brackets further down the input may already have been stored
// with a context. As soon as W7 pops up a new strong type L, those contexts
// may be invalid. The parser will detect such situations and then call this
// function to update the contexts of bracket pairs already found.
func (bph *bracketPairHandler) UpdateClosingBrackets(s scrap) {
	bph.mx.Lock()
	defer bph.mx.Unlock()
	for _, pair := range bph.pairings {
		if s.l < pair.opening.l {
			pair.opening.context.SetStrongType(s.bidiclz, s.l)
		}
		if s.l < pair.closing.l {
			pair.closing.context.SetStrongType(s.bidiclz, s.l)
		}
	}
}

func (bph *bracketPairHandler) dump() {
	if len(bph.stack) == 0 {
		tracing.Debugf("BD16: Bracket Stack is empty")
	} else {
		tracing.Debugf("BD16: Bracket Stack:")
		for i, p := range bph.stack {
			tracing.Debugf("\t[%d] %v at %d", i, p.pair, p.opening)
		}
	}
	if len(bph.pairings) == 0 {
		tracing.Debugf("BD16: No pairings found")
	} else {
		tracing.Debugf("BD16: Bracket Pairings:")
		for i, pair := range bph.pairings {
			tracing.Debugf("\t[%d] %v", i, pair)
		}
	}
}
