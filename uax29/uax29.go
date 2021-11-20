/*
Package uax29 implements Unicode Annex #29 word breaking.

Content

UAX#29 is the Unicode Annex for breaking text into graphemes, words
and sentences.
It defines code-point classes and sets of rules
for how to place break points and break inhibitors.
This file is about word breaking.

This segmenter passes all 1823 tests of the Unicode UAX#29 test suite
for word breaking.

Typical Usage

Clients instantiate a WordBreaker object and use it as the
breaking engine for a segmenter.

  onWords := uax29.NewWordBreaker()
  segmenter := uax.NewSegmenter(onWords)
  segmenter.Init(...)
  for segmenter.Next() ...

Attention

Before using word breakers, clients usually should initialize the classes and rules:

  SetupUAX29Classes()

This initializes all the code-point range tables. Initialization is
not done beforehand, as it consumes quite some memory. However, the
word breaker will call it if range tables are not yet initialized.

______________________________________________________________________

License

This project is provided under the terms of the UNLICENSE or
the 3-Clause BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: 'Unlicense' OR 'BSD-3-Clause'

You may use the project under the terms of either license.

Licenses are reproduced in the license file in the root folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package uax29

import (
	"sync"
	"unicode"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax"
	"github.com/npillmayer/uax/emoji"
)

// tracer traces to uax.segment .
func tracer() tracing.Trace {
	return tracing.Select("uax.segment")
}

// ClassForRune gets the Unicode #UAX29 word class for a Unicode code-point.
func ClassForRune(r rune) UAX29Class {
	if r == rune(0) {
		return eot
	}
	for c := UAX29Class(0); c <= ZWJClass; c++ {
		urange := rangeFromUAX29Class[c]
		if urange != nil && unicode.Is(urange, r) {
			return c
		}
	}
	return Other
}

var setupOnce sync.Once

// SetupUAX29Classes is the top-level preparation function:
// Create code-point classes for word breaking.
// Will in turn set up emoji classes as well.
// (Concurrency-safe).
//
// The word breaker will call this transparently if it has not been called beforehand.
func SetupUAX29Classes() {
	setupOnce.Do(setupUAX29Classes)
	emoji.SetupEmojisClasses()
}

// === Word Breaker ==============================================

// WordBreaker is a Breaker type used by a uax.Segmenter to break text
// up according to UAX#29 / Words.
// It implements the uax.UnicodeBreaker interface.
type WordBreaker struct {
	rules         map[UAX29Class][]uax.NfaStateFn // we manage a set of NFAs
	publisher     uax.RunePublisher               // we use the rune publishing mechanism
	longestMatch  int                             // longest active match for any rule of this word breaker
	penalties     []int                           // returned to the segmenter: penalties to insert
	weight        int                             // will multiply penalties by this factor
	previousClass UAX29Class                      // class of previously read rune
	blockedRI     bool                            // are rules for Regional_Indicator currently blocked?
}

// NewWordBreaker creates a a new UAX#29 word breaker.
//
// Usage:
//
//   onWords := NewWordBreaker()
//   segmenter := uax.NewSegmenter(onWords)
//   segmenter.Init(...)
//   for segmenter.Next() ...
//
// weight is a multiplying factor for penalties. It must be 0…w…5 and will
// be capped for values outside this range. Currently this is not used by any test and should probably left to 1.
//
func NewWordBreaker(weight int) *WordBreaker {
	gb := &WordBreaker{weight: capw(weight)}
	gb.publisher = uax.NewRunePublisher()
	gb.rules = map[UAX29Class][]uax.NfaStateFn{
		CRClass:                 {rule_NewLine},
		LFClass:                 {rule_NewLine},
		NewlineClass:            {rule_NewLine},
		ZWJClass:                {rule_WB3c, rule_WB4},
		WSegSpaceClass:          {rule_WB3d},
		ExtendClass:             {rule_WB4},
		FormatClass:             {rule_WB4},
		ALetterClass:            {rule_WB5, rule_WB6_7, rule_WB9, rule_WB13a},
		Hebrew_LetterClass:      {rule_WB5, rule_WB6_7, rule_WB7a, rule_WB7bc, rule_WB9, rule_WB13a},
		NumericClass:            {rule_WB8, rule_WB10, rule_WB11, rule_WB13a},
		ExtendNumLetClass:       {rule_WB13a, rule_WB13b},
		KatakanaClass:           {rule_WB13, rule_WB13a},
		Regional_IndicatorClass: {rule_WB15},
	}
	if rangeFromUAX29Class == nil {
		tracer().Infof("UAX#29 classes not yet initialized -> initializing")
	}
	SetupUAX29Classes()
	return gb
}

// For word breaking we need just a single emoji class.
// We append it after the last UAX#29 class, which is ZWJ.
const emojiPictographic UAX29Class = ZWJClass + 1

// CodePointClassFor returns the UAX#29 word code-point class for a rune (= code-point).
// (Interface uax.UnicodeBreaker)
func (gb *WordBreaker) CodePointClassFor(r rune) int {
	c := ClassForRune(r)
	if c == Other {
		if unicode.Is(emoji.Extended_Pictographic, r) {
			return int(emojiPictographic)
		}
	}
	return int(c)
}

// StartRulesFor starts all recognizers where the starting symbol is rune r.
// r is of code-point-class cpClass.
// (Interface uax.UnicodeBreaker)
func (gb *WordBreaker) StartRulesFor(r rune, cpClass int) {
	c := UAX29Class(cpClass)
	if c == Regional_IndicatorClass && gb.blockedRI {
		tracer().Debugf("regional indicators blocked")
		return
	}
	if rules := gb.rules[c]; len(rules) > 0 {
		tracer().P("class", c).Debugf("starting %d rule(s) for class %s", len(rules), c)
		for _, rule := range rules {
			rec := uax.NewPooledRecognizer(cpClass, rule)
			rec.UserData = gb
			rec.Expect = gb.longestMatch // we “mis-use” expect field
			gb.publisher.SubscribeMe(rec)
		}
	}
}

// Helper: do not start any recognizers for this word class, until
// unblocked again. For now, just consider regional indicators.
// We will find out if other needs arise.
func (gb *WordBreaker) block(c UAX29Class) {
	gb.blockedRI = true
}

// Helper: stop blocking new recognizers for this word class.
// For now, just consider regional indicators.
// We will find out if other needs arise.
func (gb *WordBreaker) unblock(c UAX29Class) {
	gb.blockedRI = false
}

// ProceedWithRune is a signal:
// A new code-point has been read and this breaker receives a message to
// consume it.
// (Interface uax.UnicodeBreaker)
func (gb *WordBreaker) ProceedWithRune(r rune, cpClass int) {
	c := UAX29Class(cpClass)
	tracer().P("class", c).Debugf("proceeding with rune %#U ...", r)
	gb.longestMatch, gb.penalties = gb.publisher.PublishRuneEvent(r, int(c))
	tracer().P("class", c).Debugf("...done with |match|=%d and p=%v", gb.longestMatch, gb.penalties)
	gb.previousClass = c
	setPenalty1(gb, penalty999) //gb.penalties[1] = penalty999, if empty
	//tracer().Debugf("penalites now = %v", gb.penalties)
}

// LongestActiveMatch collects
// from all active recognizers information about current match length
// and return the longest one for all still active recognizers.
// (Interface uax.UnicodeBreaker)
func (gb *WordBreaker) LongestActiveMatch() int {
	return gb.longestMatch
}

// Penalties gets all active penalties for all active recognizers combined.
// Index 0 belongs to the most recently read rune, i.e., represents
// the penalty for breaking after it.
// (Interface uax.UnicodeBreaker)
func (gb *WordBreaker) Penalties() []int {
	return gb.penalties
}

// Penalties (inter-word optional break, suppress break and mandatory break).
var (
	PenaltyForBreak        = 50
	PenaltyToSuppressBreak = 10000
	PenaltyForMustBreak    = -10000
	penalty999             = 10
)

// --- Rules ------------------------------------------------------------

func rule_NewLine(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if c == LFClass || c == NewlineClass {
		//tracer().Debugf("ACCEPT of Rule for Newline")
		return uax.DoAccept(rec, PenaltyForMustBreak, PenaltyForMustBreak)
	} else if c == CRClass {
		//tracer().Debugf("shift CR")
		rec.MatchLen++
		return rule_CRLF
	}
	return uax.DoAbort(rec)
}

func rule_CRLF(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if c == LFClass {
		//tracer().Debugf("ACCEPT of Rule for CRLF")
		return uax.DoAccept(rec, PenaltyForMustBreak, 3*PenaltyToSuppressBreak) // accept CR+LF
	}
	//tracer().Debugf("ACCEPT of Rule for CR")
	return uax.DoAccept(rec, 0, PenaltyForMustBreak, PenaltyForMustBreak) // accept CR
}

func rule_WB3c(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	rec.MatchLen++
	return rule_Pictography
}

func rule_Pictography(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if c == emojiPictographic {
		//tracer().Debugf("ACCEPT of Rule for Emoji")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

func rule_WB3d(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 3d")
	rec.MatchLen++
	return finish_WB3d
}

func finish_WB3d(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("WB3d cont")
	c := UAX29Class(cpClass)
	if c == WSegSpaceClass {
		//tracer().Debugf("ACCEPT of Rule WB 3d")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

func checkIgnoredCharacters(rec *uax.Recognizer, c UAX29Class) bool {
	if c == ExtendClass || c == FormatClass || c == ZWJClass {
		rec.MatchLen++
		return true
	}
	return false
}

// start AHLetter x AHLetter
func rule_WB5(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 5")
	rec.MatchLen++
	return finish_WB5_10
}

// ... x AHLetter
func finish_WB5_10(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB5_10
	}
	if c == ALetterClass || c == Hebrew_LetterClass {
		//tracer().Debugf("ACCEPT of Rule WB 5/10")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start AHLetter x (MidLetter | MidNumLet | Single_Quote) x AHLetter
func rule_WB6_7(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debugf("start WB 6/7")
	rec.MatchLen++
	return cont_WB6_7
}

// ... x  (MidLetter | MidNumLet | Single_Quote) x AHLetter
func cont_WB6_7(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return cont_WB6_7
	}
	if c == MidLetterClass || c == MidNumLetClass || c == Single_QuoteClass {
		//tracer().Debugf("MID LETTER IN RULE 6/7 cont 1")
		rec.MatchLen++
		rec.Expect = rec.MatchLen // misuse of expect field: mark position of single quote
		return finish_WB6_7
	}
	return uax.DoAbort(rec)
}

// ... x ... x AHLetter
func finish_WB6_7(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB6_7
	}
	if c == ALetterClass || c == Hebrew_LetterClass {
		//tracer().Debugf("ACCEPT of Rule WB 6/7")
		p := make([]int, rec.MatchLen-rec.Expect+1+2)
		p[len(p)-1] = PenaltyToSuppressBreak
		p[1] = PenaltyToSuppressBreak
		return uax.DoAccept(rec, p...)
	}
	return uax.DoAbort(rec)
}

// start Hebrew_Letter x Single_Quote
func rule_WB7a(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debugf("start WB 7a")
	rec.MatchLen++
	return finish_WB7a
}

// ... x Single_Quote
func finish_WB7a(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB7a
	}
	if c == Single_QuoteClass {
		//tracer().Debugf("ACCEPT of Rule WB 7 a")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start Hebrew_Letter x Double_Quote x Hebrew_Letter
func rule_WB7bc(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debugf("start WB 7c")
	rec.MatchLen++
	return cont_WB7bc
}

func cont_WB7bc(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return cont_WB7bc
	}
	if c == Double_QuoteClass {
		rec.MatchLen++
		return finish_WB7bc
	}
	return uax.DoAbort(rec)
}

func finish_WB7bc(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB7bc
	}
	if c == Hebrew_LetterClass {
		//tracer().Debugf("ACCEPT of Rule WB 7b,c")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start Numeric x Numeric
func rule_WB8(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 8")
	rec.MatchLen++
	return finish_WB8_9
}

// start (ALetter | Hebrew_Letter) x Numeric
func rule_WB9(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 9")
	rec.MatchLen++
	return finish_WB8_9
}

// ... x Numeric
func finish_WB8_9(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB8_9
	}
	if c == NumericClass {
		//tracer().Debugf("ACCEPT of Rule WB 8/9")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start Numeric x AHLetter
func rule_WB10(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 10")
	rec.MatchLen++
	return finish_WB5_10
}

// start Numeric x (MidNum | MidNumLet | Single_Quote) x Numeric
func rule_WB11(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debugf("start WB 11")
	rec.MatchLen++
	return cont_WB11
}

// ... x (MidNum | MidNumLet | Single_Quote) x Numeric
func cont_WB11(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return cont_WB11
	}
	if c == MidNumClass || c == MidNumLetClass || c == Single_QuoteClass {
		rec.MatchLen++
		rec.Expect = rec.MatchLen // misuse of expect field: mark position of middle character
		//tracer().Debugf("continue WB 11")
		return finish_WB11
	}
	return uax.DoAbort(rec)
}

// ... x ... x Numeric
func finish_WB11(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB11
	}
	if c == NumericClass {
		//tracer().Debugf("ACCEPT of Rule WB 11")
		p := make([]int, rec.MatchLen-rec.Expect+1+2)
		p[len(p)-1] = PenaltyToSuppressBreak
		p[1] = PenaltyToSuppressBreak
		return uax.DoAccept(rec, p...)
		//return uax.DoAccept(rec, 0, PenaltyToSuppressBreak, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start Katakana x Katakana
func rule_WB13(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 13")
	rec.MatchLen++
	return finish_WB13
}

// ... x Katakana
func finish_WB13(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB13
	}
	if c == KatakanaClass {
		//tracer().Debugf("ACCEPT of Rule WB 13")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start (AHLetter | Numeric | Katakana | ExtendNumLet) x ExtendNumLet
func rule_WB13a(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 13a")
	rec.MatchLen++
	return finish_WB13a
}

// ... x ExtendNumLet
func finish_WB13a(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB13a
	}
	if c == ExtendNumLetClass {
		//tracer().Debugf("ACCEPT of Rule WB 13 a")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start ExtendNumLet x (AHLetter | Numeric | Katakana)
func rule_WB13b(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 13b")
	rec.MatchLen++
	return finish_WB13b
}

// ... x (AHLetter | Numeric | Katakana)
func finish_WB13b(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB13b
	}
	if c == ALetterClass || c == Hebrew_LetterClass || c == NumericClass || c == KatakanaClass {
		//tracer().Debugf("ACCEPT of Rule WB 13 b")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// start RI x RI (blocking)
func rule_WB15(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debug("start WB 15")
	rec.MatchLen++
	gb := rec.UserData.(*WordBreaker)
	gb.block(Regional_IndicatorClass)
	return finish_WB15
}

// ... x RI
func finish_WB15(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := UAX29Class(cpClass)
	if checkIgnoredCharacters(rec, c) {
		return finish_WB15
	}
	gb := rec.UserData.(*WordBreaker)
	gb.unblock(Regional_IndicatorClass)
	if c == Regional_IndicatorClass {
		//tracer().Debugf("ACCEPT of Rule WB 15")
		return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
	}
	return uax.DoAbort(rec)
}

// Rule WB4: Ignore Format and Extend characters, except after sot, CR,
// LF, and Newline.
func rule_WB4(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//tracer().Debugf("start WB 4")
	c := UAX29Class(cpClass)
	if c == ExtendClass || c == FormatClass || c == ZWJClass {
		gb := rec.UserData.(*WordBreaker)
		prev := gb.previousClass
		if prev != LFClass && prev != NewlineClass && prev != CRClass {
			//tracer().Debugf("ACCEPT of Rule WB 4")
			return uax.DoAccept(rec, 0, PenaltyToSuppressBreak)
		}
	}
	return uax.DoAbort(rec)
}

// --- Helpers ---------------------------------------------------------------
func packPenalties(rec *uax.Recognizer, penalties ...int) []int {
	return penalties
}

func setPenalty1(gb *WordBreaker, p int) {
	if len(gb.penalties) == 0 {
		gb.penalties = append(gb.penalties, 0)
		gb.penalties = append(gb.penalties, p)
	} else if len(gb.penalties) == 1 {
		gb.penalties = append(gb.penalties, p)
	} else if gb.penalties[1] == 0 {
		gb.penalties[1] = p
	}
}

func extendFormat(c UAX29Class) bool {
	return c == ExtendClass || c == FormatClass
}

func extendFormatZWJ(c UAX29Class) bool {
	return c == ExtendClass || c == FormatClass || c == ZWJClass
}

func capw(w int) int {
	if w < 0 {
		return w
	}
	if w > 5 {
		return 5
	}
	return w
}
