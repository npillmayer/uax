/*
Package uax14 implements Unicode Annex #14 line breaking.

Under active development; use at your own risk

BSD License

Copyright (c) 2017-20, Norbert Pillmayer

All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


Contents

UAX#14 is the Unicode Annex for Line Breaking (Line Wrap).
It defines a bunch of code-point classes and a set of rules
for how to place break points / break inhibitors.

Typical Usage

Clients instantiate a UAX#14 line breaker object and use it as the
breaking engine for a segmenter.

  breaker := uax14.NewLineWrap()
  segmenter := unicode.NewSegmenter(breaker)
  segmenter.Init(...)
  for segmenter.Next() {
    ... // do something with segmenter.Text() or segmenter.Bytes()
  }

Before using line breakers, clients usually will want to initialize the UAX#14
classes and rules.

  SetupClasses()

This initializes all the code-point range tables. Initialization is
not done beforehand, as it consumes quite some memory, and using UAX#14
is not mandatory. SetupClasses() is called automatically, however,
if clients call NewLineWrap().

Status

The current implementation does not pass all tests from the UAX#14
test file. The reason is the interpretation of rules involving ZWJ.
I am a bit at a loss about ZWJ rules, as from my
point of view they aren't consistent.

	=== RUN   TestWordBreakTestFile
	--- FAIL: TestWordBreakTestFile (0.36s)
		uax14_test.go:54: test #6263: '"\u200d"' should be '"\u200d\u231a"'
		...
		uax14_test.go:54: test #6339: '"\u200d"' should be '"\u200d\u261d"'
		...
		uax14_test.go:54: test #6343: '"\u200d"' should be '"\u200d\U0001f3fb"'
		...
		uax14_test.go:43: 3 TEST CASES OUT of 7282 FAILED
	FAIL

3 out of the 7282 test cases fail. That doesn't sound too much of a problem,
but I'm afraid the interpretation I chose is not the best one. Nevertheless,
at this point I'm inclined to postpone the problem and to first seek some
practical experience with real-life multi-lingual texts. */
package uax14

import (
	"math"
	"sync"
	"unicode"

	"github.com/npillmayer/schuko/gtrace"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax"
)

const (
	sot       UAX14Class = 1000 // pseudo class
	eot       UAX14Class = 1001 // pseudo class
	optSpaces UAX14Class = 1002 // pseudo class
)

// TC traces to the core-tracer.
func TC() tracing.Trace {
	return gtrace.CoreTracer
}

// ClassForRune is the top-level client function:
// Get the line breaking/wrap class for a Unicode code-point
func ClassForRune(r rune) UAX14Class {
	if r == rune(0) {
		return eot
	}
	for lbc := UAX14Class(0); lbc <= ZWJClass; lbc++ {
		urange := rangeFromUAX14Class[lbc]
		if urange == nil {
			TC().Errorf("-- no range for class %s\n", lbc)
		} else if unicode.Is(urange, r) {
			return lbc
		}
	}
	return XXClass
}

var setupOnce sync.Once

// SetupClasses is the top-level preparation function:
// Create code-point classes for UAX#14 line breaking/wrap.
// (Concurrency-safe).
func SetupClasses() {
	setupOnce.Do(setupUAX14Classes)
}

// === UAX#14 Line Breaker ==============================================

// LineWrap is a type used by a unicode.Segmenter to break lines
// up according to UAX#14. It implements the unicode.UnicodeBreaker interface.
type LineWrap struct {
	publisher    uax.RunePublisher
	longestMatch int   // longest active match of a rule
	penalties    []int // returned to the segmenter: penalties to insert
	rules        map[UAX14Class][]uax.NfaStateFn
	lastClass    UAX14Class // we have to remember the last code-point class
	blockedRI    bool       // are rules for Regional_Indicator currently blocked?
	substituted  bool       // has the code-point class been substituted?
	shadow       UAX14Class // class before substitution
}

// NewLineWrap creates a new UAX#14 line breaker.
//
// Usage:
//
//   linewrap := NewLineWrap()
//   segmenter := segment.NewSegmenter(linewrap)
//   segmenter.Init(...)
//   for segmenter.Next() ...
//
func NewLineWrap() *LineWrap {
	uax14 := &LineWrap{}
	uax14.publisher = uax.NewRunePublisher()
	uax14.rules = map[UAX14Class][]uax.NfaStateFn{
		//sot:      {rule_LB2},
		NLClass:  {rule_05_NewLine},
		LFClass:  {rule_05_NewLine},
		BKClass:  {rule_05_NewLine},
		CRClass:  {rule_05_NewLine},
		SPClass:  {rule_LB7, rule_LB18},
		ZWClass:  {rule_LB7, rule_LB8},
		WJClass:  {rule_LB11},
		GLClass:  {rule_LB12},
		CLClass:  {rule_LB13, rule_LB16},
		CPClass:  {rule_LB13, rule_LB16, rule_LB30_2},
		EXClass:  {rule_LB13, rule_LB22},
		ISClass:  {rule_LB13, rule_LB29},
		SYClass:  {rule_LB13, rule_LB21b},
		OPClass:  {rule_LB14, step2_LB25},
		QUClass:  {rule_LB15, rule_LB19},
		B2Class:  {rule_LB17},
		BAClass:  {rule_LB21},
		CBClass:  {rule_LB20},
		HYClass:  {rule_LB21, step2_LB25},
		NSClass:  {rule_LB21},
		BBClass:  {rule_LB21x},
		ALClass:  {rule_LB22, rule_LB23_1, rule_LB24_2, rule_LB28, rule_LB30_1},
		HLClass:  {rule_LB21a, rule_LB22, rule_LB23_1, rule_LB24_2, rule_LB28, rule_LB30_1},
		IDClass:  {rule_LB22, rule_LB23a_2},
		EBClass:  {rule_LB22, rule_LB23a_2, rule_LB30b},
		EMClass:  {rule_LB22, rule_LB23a_2},
		INClass:  {rule_LB22},
		NUClass:  {rule_LB22, rule_LB23_2, step3_LB25, rule_LB30_1},
		RIClass:  {rule_LB30a},
		PRClass:  {rule_LB23a_1, rule_LB24_1, rule_LB25, rule_LB27_2},
		POClass:  {rule_LB24_1, rule_LB25},
		JLClass:  {rule_LB26_1, rule_LB27},
		JVClass:  {rule_LB26_2, rule_LB27},
		H2Class:  {rule_LB26_2, rule_LB27},
		JTClass:  {rule_LB26_3, rule_LB27},
		H3Class:  {rule_LB26_3, rule_LB27},
		ZWJClass: {rule_LB8a},
	}
	if rangeFromUAX14Class == nil {
		TC().Infof("UAX#14 classes not yet initialized -> initializing")
	}
	SetupClasses()
	uax14.lastClass = sot
	return uax14
}

// CodePointClassFor returns the UAX#14 code-point class for a rune (= code-point).
//
// Interface unicode.UnicodeBreaker
func (uax14 *LineWrap) CodePointClassFor(r rune) int {
	c := ClassForRune(r)
	c = resolveSomeClasses(r, c)
	cnew, shadow := substitueSomeClasses(c, uax14.lastClass)
	uax14.substituted = (c != cnew)
	uax14.shadow = shadow
	return int(cnew)
}

// StartRulesFor starts all recognizers where the starting symbol is rune r.
// r is of code-point-class cpClass.
//
// Interface unicode.UnicodeBreaker
func (uax14 *LineWrap) StartRulesFor(r rune, cpClass int) {
	c := UAX14Class(cpClass)
	if c != RIClass || !uax14.blockedRI {
		if rules := uax14.rules[c]; len(rules) > 0 {
			TC().P("class", c).Debugf("starting %d rule(s) for class %s", len(rules), c)
			for _, rule := range rules {
				rec := uax.NewPooledRecognizer(cpClass, rule)
				rec.UserData = uax14
				uax14.publisher.SubscribeMe(rec)
			}
		} else {
			TC().P("class", c).Debugf("starting no rule")
		}
		/*
			if uax14.shadow == ZWJClass {
				if rules := uax14.rules[uax14.shadow]; len(rules) > 0 {
					TC.P("class", c).Debugf("starting %d rule(s) for shadow class ZWJ", len(rules))
					for _, rule := range rules {
						rec := uax.NewPooledRecognizer(cpClass, rule)
						rec.UserData = uax14
						uax14.publisher.SubscribeMe(rec)
					}
				}
			}
		*/
	}
}

// LB1 Assign a line breaking class to each code point of the input.
// Resolve AI, CB, CJ, SA, SG, and XX into other line breaking classes
// depending on criteria outside the scope of this algorithm.
//
// In the absence of such criteria all characters with a specific combination of
// original class and General_Category property value are resolved as follows:
//
//   Resolved 	Original 	 General_Category
//   AL         AI, SG, XX  Any
//   CM         SA          Only Mn or Mc
//   AL         SA          Any except Mn and Mc
//   NS         CJ          Any
//
func resolveSomeClasses(r rune, c UAX14Class) UAX14Class {
	if c == AIClass || c == SGClass || c == XXClass {
		return ALClass
	} else if c == SAClass {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) {
			return CMClass
		}
		return ALClass
	} else if c == CJClass {
		return NSClass
	}
	return c
}

// LB9: Do not break a combining character sequence;
// treat it as if it has the line breaking class of the base character in all
// of the following rules. Treat ZWJ as if it were CM.
//
//    X (CM | ZWJ)* ⟼ X.
//
// where X is any line break class except BK, CR, LF, NL, SP, or ZW.
//
// LB10: Treat any remaining combining mark or ZWJ as AL.
func substitueSomeClasses(c UAX14Class, lastClass UAX14Class) (UAX14Class, UAX14Class) {
	shadow := c
	switch lastClass {
	case sot, BKClass, CRClass, LFClass, NLClass, SPClass, ZWClass:
		if c == CMClass || c == ZWJClass {
			c = ALClass
		}
	default:
		if c == CMClass || c == ZWJClass {
			c = lastClass
		}
	}
	if shadow != c {
		TC().Debugf("subst %+q -> %+q", shadow, c)
	}
	return c, shadow
}

// ProceedWithRune is part of interface unicode.Breaker.
// A new code-point has been read and this breaker receives a message to
// consume it.
func (uax14 *LineWrap) ProceedWithRune(r rune, cpClass int) {
	c := UAX14Class(cpClass)
	uax14.longestMatch, uax14.penalties = uax14.publisher.PublishRuneEvent(r, int(c))
	x := uax14.penalties
	//fmt.Printf("   x = %v\n", x)
	if uax14.substituted && uax14.lastClass == c { // do not break between runes for rule 09
		if len(x) > 1 && x[1] == 0 {
			x[1] = 1000
		} else if len(x) == 1 {
			x = append(x, 1000)
		} else if len(x) == 0 {
			x = make([]int, 2)
			x[1] = 1000
		}
	}
	for i, p := range x { // positive penalties get lifted +1000
		if p > 0 {
			p += 1000
			x[i] = p
		}
	}
	//fmt.Printf("=> x = %v\n", x)
	uax14.penalties = x
	if c == eot { // start all over again
		c = sot
	}
	uax14.lastClass = c
}

// LongestActiveMatch is part of interface unicode.UnicodeBreaker
func (uax14 *LineWrap) LongestActiveMatch() int {
	return uax14.longestMatch
}

// Penalties gets all active penalties for all active recognizers combined.
// Index 0 belongs to the most recently read rune.
//
// Interface unicode.UnicodeBreaker
func (uax14 *LineWrap) Penalties() []int {
	return uax14.penalties
}

// Helper: do not start any recognizers for class RI, until
// unblocked again.
func (uax14 *LineWrap) block() {
	uax14.blockedRI = true
}

// Helper: stop blocking new recognizers for class RI.
func (uax14 *LineWrap) unblock() {
	uax14.blockedRI = false
}

// Penalties (suppress break and mandatory break).
var (
	PenaltyToSuppressBreak = 5000
	PenaltyForMustBreak    = -10000
)

// This is a small function to return a penalty value for a rule.
// w is the weight of the rule (currently I use the rule number
// directly).
func p(w int) int {
	q := 31 - w
	r := int(math.Pow(1.3, float64(q)))
	TC().P("rule", w).Debugf("penalty %d => %d", w, r)
	return r
}

// Helper to create a slice of integer penalties, usually of length
// MatchLen for an accepting rule.
func ps(w int, first int, l int) []int {
	pp := make([]int, l+1)
	pp[1] = first
	for i := 2; i <= l; i++ {
		pp[i] = p(w)
	}
	return pp
}
