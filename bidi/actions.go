package bidi

import (
	"strings"

	"github.com/npillmayer/uax/internal/tracing"
	"golang.org/x/text/unicode/bidi"
)

// We create a set of bidi rules as layed out in UAX#9.
// To understand the rules it is probably best to consult the UAX algorithm
// description. Headers and rule names will be similar to names in UAX#9.
//
// Rules are structs which contain an action function. The rule's struct
// has an left-hand-side (LHS) part that will be matched against the input text
// (more specific: against bidi class clusters = scraps) and the rule's action will
// substitute the LHS with an appropriate RHS.
//
// Invariants:
// - The RHS must be of equal length to the LHS or shorter.
// - No LHS must be prefix of another rule's LHS, except for |LHS|=1 .
// - Every rule is required to either have RHS≠LHS or to return a jump offset ≠ 0,
//   otherwise the parser will enter an infinite loop.
//
// All rules are hand-checked to not create circular identities and therefore
// are guaranteed to not send the parser into an infinite loop.
//
type bidiRule struct {
	name   string     // name of the rule according to UAX#9
	lhsLen int        // number of symbols in the left hand side (LHS)
	pass   int        // this is a 2-pass system, every rule is active during just one phase
	action ruleAction // action to perform on match of LHS
}

// ruleAction is an action on bidi class intervals. Input is a slice of (consecutive)
// class intervals which have been matched. The action's task is to substitute all or some
// of the input intervals by one or more output intervals (reduce action). The ``cursor''
// will be positioned after the substitution by the parser, according to the second result
// of the action, an integer. This position hint will be negative most of the time, telling
// the parser to backtrack and try to re-apply other BiDi rules.
type ruleAction func([]scrap) ([]scrap, int, bool)

// Headers and header numbers of the following comment sections correspond to UAX#9.

// ---------------------------------------------------------------------------
// 3.3.4 Resolving Weak Types

// W1 – W3 are handled by the scanner.

// W1. Examine each nonspacing mark (NSM) in the isolating run sequence, and change the
//     type of the NSM to Other Neutral if the previous character is an isolate
//     initiator or PDI, and to the type of the previous character otherwise.

// W2. Search backward from each instance of a European number until the first strong type
//     (R, L, AL, or sos) is found. If an AL is found, change the type of the
//     European number to Arabic number.
// AL EN     → AL AN
// AL NI EN  → AL NI AN

// W3. Change all ALs to R.

// --- Actions ---

// W4. A single European separator between two European numbers changes to
//     a European number. A single common separator between two numbers of the
//     same type changes to that type.
// EN ES EN → EN EN EN
// EN CS EN → EN EN EN
// AN CS AN → AN AN AN

func ruleW4_1() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN, bidi.ES, bidi.EN)
	return makeSquashRule("W4-1", lhs, bidi.EN, -2), lhs
}

func ruleW4_2() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN, bidi.CS, bidi.EN)
	return makeSquashRule("W4-2", lhs, bidi.EN, -2), lhs
}

func ruleW4_3() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.AN, bidi.CS, bidi.AN)
	return makeSquashRule("W4-3", lhs, bidi.AN, -2), lhs
}

// W5. A sequence of European terminators adjacent to European numbers
//     changes to all European numbers.
// ET ET EN → EN EN EN
// EN ET ET → EN EN EN
// AN ET EN → AN EN EN

func ruleW5_1() (*bidiRule, []byte) { // W5-1 and W5-3
	lhs := makeLHS(bidi.ET, bidi.EN)
	return makeSquashRule("W5-1", lhs, bidi.EN, -2), lhs
}

func ruleW5_2() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN, bidi.ET)
	return makeSquashRule("W5-2", lhs, bidi.EN, -2), lhs
}

// W6. Otherwise, separators and terminators change to Other Neutral.
// AN ET    → AN ON
// L  ES EN → L  ON EN
// EN CS AN → EN ON AN
// ET AN    → ON AN

func ruleW6_1() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.ET)
	return makeSquashRule("W6-1", lhs, cNI, 0), lhs
}

func ruleW6_2() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.ES)
	return makeSquashRule("W6-2", lhs, cNI, 0), lhs
}

func ruleW6_3() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.CS)
	return makeSquashRule("W6-3", lhs, cNI, 0), lhs
}

func ruleW6x() (*bidiRule, []byte) {
	lhs := makeLHS(cNI, cNI)
	return makeSquashRule("W6-x", lhs, cNI, -1), lhs
}

// W7. Search backward from each instance of a European number until the
// first strong type (R, L, or sos) is found. If an L is found, then change the
// type of the European number to L.
// L  NI EN → L  NI  L
// R  NI EN → R  NI  EN

func ruleW7() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN)
	return &bidiRule{
		name:   "W7",
		lhsLen: len(lhs),
		pass:   1,
		action: func(match []scrap) ([]scrap, int, bool) {
			if match[0].Context() == bidi.L {
				L := match[:1]
				L[0].bidiclz = bidi.L
				return L, 0, true // replace EN with L, start again with L
			}
			return match, 1, true // step over EN
		},
	}, lhs
}

// ---------------------------------------------------------------------------
// 3.3.5 Resolving Neutral and Iolate Formatting Types

// func ruleN1_0() (*bidiRule, []byte) {
// 	lhs := makeLHS(cNI, cNI)
// 	return makeSquashRule("N1-0", lhs, bidi.L, 0), lhs
// }

// N1. A sequence of NIs takes the direction of the surrounding strong text if the text
//     on both sides has the same direction. European and Arabic numbers act as if they
//     were R in terms of their influence on NIs. The start-of-sequence (sos) and
//     end-of-sequence (eos) types are used at isolating run sequence boundaries.
// L  NI   L  →   L  L   L    (1)
// R  NI   R  →   R  R   R    (2)
// R  NI  AN  →   R  R  AN    (3)
// R  NI  EN  →   R  R  EN    (4)
// AN  NI   R  →  AN  R   R   (5)
// AN  NI  AN  →  AN  R  AN   (6)
// AN  NI  EN  →  AN  R  EN   (7)
// EN  NI   R  →  EN  R   R   (8)
// EN  NI  AN  →  EN  R  AN   (9)
// EN  NI  EN  →  EN  R  EN   (10)

func ruleN1_1() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.L, cNI, bidi.L)
	return makeSquashRule("N1-1", lhs, bidi.L, 0), lhs
}

func ruleN1_2() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.R, cNI, bidi.R)
	return makeSquashRule("N1-2", lhs, bidi.R, 0), lhs
}

func ruleN1_3() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.R, cNI, bidi.AN) // R NI → R
	return &bidiRule{
		name:   "N1-3",
		lhsLen: len(lhs),
		pass:   2,
		action: func(match []scrap) ([]scrap, int, bool) {
			collapse(match[0], match[1], bidi.R)
			match[1].bidiclz = bidi.AN
			match[1].appendAllChildrenOf(match[2])
			return match[:2], 1, false
		},
	}, lhs
}

func ruleN1_4() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.R, cNI, bidi.EN) // R NI → R
	return &bidiRule{
		name:   "N1-4",
		lhsLen: len(lhs),
		pass:   2,
		action: func(match []scrap) ([]scrap, int, bool) {
			collapse(match[0], match[1], bidi.R)
			match[1].bidiclz = bidi.EN
			match[1].appendAllChildrenOf(match[2])
			return match[:2], 1, false
		},
	}, lhs
}

func ruleN1_5() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.AN, cNI, bidi.R) // NI R → R
	return &bidiRule{
		name:   "N1-5",
		lhsLen: len(lhs),
		pass:   2,
		action: func(match []scrap) ([]scrap, int, bool) {
			collapse(match[1], match[2], bidi.R)
			return match[:2], 1, false
		},
	}, lhs
}

func ruleN1_6() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.AN, cNI, bidi.AN) // NI → R
	return makeMidSwapRule("N1-6", lhs, bidi.R, 2), lhs
}

func ruleN1_7() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.AN, cNI, bidi.EN) // NI → R
	return makeMidSwapRule("N1-7", lhs, bidi.R, 2), lhs
}

func ruleN1_8() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN, cNI, bidi.R) // NI R → R
	return &bidiRule{
		name:   "N1-8",
		lhsLen: len(lhs),
		pass:   2,
		action: func(match []scrap) ([]scrap, int, bool) {
			collapse(match[1], match[2], bidi.R)
			return match[:2], 1, false
		},
	}, lhs
}

func ruleN1_9() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN, cNI, bidi.AN) // NI → R
	return makeMidSwapRule("N1-9", lhs, bidi.R, 2), lhs
}

func ruleN1_10() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.EN, cNI, bidi.EN) // NI → R
	return makeMidSwapRule("N1-10", lhs, bidi.R, 2), lhs
}

// N2. Any remaining NIs take the embedding direction.
// NI → e
func ruleN2() (*bidiRule, []byte) {
	lhs := makeLHS(cNI) // NI → e
	return &bidiRule{
		name:   "N2",
		lhsLen: len(lhs),
		pass:   2,
		action: func(match []scrap) ([]scrap, int, bool) {
			ni := match[0]
			ni.bidiclz = ni.e()
			tracing.Debugf("rule N2: produced e=%s with context=%v", ni, ni.context)
			return []scrap{ni}, -1, false
		},
	}, lhs
}

func ruleL() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.L, bidi.L)
	return makeSquashRule("L+L=L", lhs, bidi.L, 0), lhs
}

func ruleR() (*bidiRule, []byte) {
	lhs := makeLHS(bidi.R, bidi.R)
	return makeSquashRule("R+R=R", lhs, bidi.R, 0), lhs
}

// ---------------------------------------------------------------------------

func makeSquashRule(name string, lhs []byte, c bidi.Class, jmp int) *bidiRule {
	r := &bidiRule{
		name:   name,
		lhsLen: len(lhs),
		action: squash(c, len(lhs), jmp),
	}
	if strings.HasPrefix(name, "W") {
		r.pass = 1
	} else {
		r.pass = 2
	}
	return r
}

func squash(c bidi.Class, n int, jmp int) ruleAction {
	return func(match []scrap) ([]scrap, int, bool) {
		last := match[n-1]
		//T().Debugf("squash: match=%v,  last = %s", match, last)
		match[0].r = last.r
		match[0].bidiclz = c
		for i, iv := range match[:n] {
			if i == 0 {
				continue
			}
			match[0].appendAllChildrenOf(iv)
		}
		return match[:1], jmp, false
	}
}

func makeMidSwapRule(name string, lhs []byte, c bidi.Class, jmp int) *bidiRule {
	return &bidiRule{
		name:   name,
		lhsLen: len(lhs),
		pass:   2, // all mid-swap rules are Nx rules ⇒ pass 2
		action: func(match []scrap) ([]scrap, int, bool) {
			match[1].bidiclz = c // change class of middle interval
			return match, jmp, false
		},
	}
}

func makeLHS(toks ...bidi.Class) []byte {
	b := make([]byte, len(toks))
	for i, t := range toks {
		b[i] = byte(t)
	}
	return b
}

func appendChildren(dest scrap, src scrap) {
	tracing.Errorf("appendChildren(…) not yet implemented")
}
