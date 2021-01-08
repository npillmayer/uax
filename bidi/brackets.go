package bidi

import (
	"fmt"
	"sync"

	"golang.org/x/text/unicode/bidi"
)

// BD16MaxNesting is the maximum stack depth for rule BS16 as defined in UAX#9.
const BD16MaxNesting = 63

// --- Brackets and bracket stack --------------------------------------------

type bracketPairHandler struct {
	stack    bracketStack        // stack to trace bracket nesting
	pairings pairingsList        // result list of matching bracket pairings
	mx       sync.Mutex          // guards pairings, as parser will access them, too
	outer    *bracketPairHandler // handlers will be stacked for nested isolate runs
}

type bracketStack []brktpos
type brktpos struct {
	pos  charpos // position of an opening bracket
	pair bracketPair
}

type pairingsList []pairing
type pairing struct {
	brktpos
	closing scrap // closing bracket represented as a scrap
}

func makeBracketPairHandler(outer *bracketPairHandler) *bracketPairHandler {
	h := &bracketPairHandler{
		stack:    make(bracketStack, 0, BD16MaxNesting),
		pairings: make(pairingsList, 0, 8),
		outer:    outer,
	}
	return h
}

func (pr pairing) String() string {
	return fmt.Sprintf("[%#U,%#U] at %d → %s", pr.pair.o, pr.pair.c, pr.pos, pr.closing)
}

func (bph *bracketPairHandler) pushOpening(r rune, s scrap) bool {
	var ok bool
	ok, bph.stack = bph.stack.push(r, s.bidiclz, s.l)
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
		if newPairing.pos < pair.pos {
			inx = i
		}
	}
	bph.pairings = append(bph.pairings[:inx], append([]pairing{newPairing}, bph.pairings[inx:]...)...)
	return
}

func (bs bracketStack) push(r rune, bidiclz bidi.Class, pos charpos) (bool, bracketStack) {
	// TODO BS16Max is the limit per isolating run sequence, not overall
	if len(bs) >= BD16MaxNesting { // skip in case of stack overflow, as defined in UAX#9
		return false, bs
	}
	if bidiclz != BRACKO {
		return false, bs
	}
	// TODO put bracket list in sutable data structure (map like) ?
	for _, pair := range uax9BracketPairs { // double check for UAX#9 brackets
		if pair.o == r {
			b := brktpos{pos: pos, pair: pair}
			return true, append(bs, b)
		}
	}
	T().Errorf("Push of %c failed, not found as opening bracket")
	return false, bs
}

// func (bs bracketStack) pushIfBracket(b rune, pos uint64) (bool, bracketStack) {
// 	props, _ := bidi.LookupRune(b)
// 	if props.IsBracket() && props.IsOpeningBracket() {
// 		T().Errorf("pushing bracket %v, bracket stack was %v", b, bs)
// 		return bs.push(b, pos)
// 	}
// 	return false, bs
// }

func (bs bracketStack) popWith(b rune, pos charpos) (bool, brktpos, bracketStack) {
	T().Debugf("popWith: rune=%v, bracket stack is %v", b, bs)
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

// const (
// 	Opening bool = true
// 	Closing bool = false
// )

//func (bph *bracketPairHandler) FindBracketPairing(s scrap, open bool) (pairing, bool) {
func (bph *bracketPairHandler) FindBracketPairing(s scrap) (pairing, bool) {
	bph.mx.Lock()
	defer bph.mx.Unlock()
	for _, pair := range bph.pairings {
		if s.bidiclz == BRACKO { // scrap is opening bracket
			if pair.pos == s.l {
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

func (bph *bracketPairHandler) UpdateClosingBrackets(bidiclz bidi.Class, pos charpos) {
	bph.mx.Lock()
	defer bph.mx.Unlock()
	for _, pair := range bph.pairings {
		lpos, rpos := pair.closing.strong.LRPos()
		if bidiclz == bidi.L && charpos(lpos) < pos && pos < pair.closing.l {
			pair.closing.strong.SetLDist(pos)
		} else if bidiclz == bidi.R && charpos(rpos) < pos && pos < pair.closing.l {
			pair.closing.strong.SetRDist(pos)
		}
	}
}

func isbracket(s scrap) bool {
	clz := s.bidiclz
	return clz == BRACKO || clz == BRACKC
}

func (bph *bracketPairHandler) dump() {
	if len(bph.stack) == 0 {
		T().Debugf("BD16: Bracket Stack is empty")
	} else {
		T().Debugf("BD16: Bracket Stack:")
		for i, p := range bph.stack {
			T().Debugf("\t[%d] %v at %d", i, p.pair, p.pos)
		}
	}
	if len(bph.pairings) == 0 {
		T().Debugf("BD16: No pairings found")
	} else {
		T().Debugf("BD16: Bracket Pairings:")
		for i, pair := range bph.pairings {
			T().Debugf("\t[%d] %v", i, pair)
		}
	}
}
