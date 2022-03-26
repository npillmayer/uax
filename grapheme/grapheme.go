package grapheme

/*
BSD License

Copyright (c) 2017–21, Norbert Pillmayer

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
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRETC, INDIRETC, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRATC, STRITC LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*/

import (
	"sync"
	"unicode"

	"github.com/npillmayer/uax"
	"github.com/npillmayer/uax/emoji"
	"github.com/npillmayer/uax/internal/tracing"
)

// ClassForRune gets the line grapheme class for a Unicode code-point.
func ClassForRune(r rune) GraphemeClass {
	if r == rune(0) {
		return eot
	}
	for c := GraphemeClass(0); c <= ZWJClass; c++ {
		urange := rangeFromGraphemeClass[c]
		if urange != nil && unicode.Is(urange, r) {
			return c
		}
	}
	return Any
}

var setupOnce sync.Once

// SetupGraphemeClasses is the top-level preparation function:
// Create code-point classes for grapheme breaking.
// Will in turn set up emoji classes as well.
// (Concurrency-safe).
func SetupGraphemeClasses() {
	setupOnce.Do(setupGraphemeClasses)
}

// === Grapheme Breaker ==============================================

// Breaker is a type to be used by a uax.Segmenter to break text
// up according to UAX#29 / Graphemes.
// It implements the uax.UnicodeBreaker interface.
type Breaker struct {
	publisher    uax.RunePublisher
	longestMatch int
	penalties    []int
	rules        map[GraphemeClass][]uax.NfaStateFn
	emojirules   map[int][]uax.NfaStateFn
	blocked      map[GraphemeClass]bool
	weight       int
}

// NewBreaker creates a new UAX#29 line breaker.
//
// Usage:
//
//   onGraphemes := NewBreaker()
//   segmenter := uax.NewSegmenter(onGraphemes)
//   segmenter.Init(...)
//   for segmenter.Next() ...
//
// weight is a multilying factor for penalties. It must be 0…w…5 and will
// be capped for values outside this range.
//
func NewBreaker(weight int) *Breaker {
	gb := &Breaker{weight: capw(weight)}
	gb.publisher = uax.NewRunePublisher()
	//gb.publisher.SetPenaltyAggregator(uax.MaxPenalties)
	gb.rules = map[GraphemeClass][]uax.NfaStateFn{
		//eot:                   {rule_GB2},
		CRClass:                 {rule_NewLine},
		LFClass:                 {rule_NewLine},
		ControlClass:            {rule_Control},
		LClass:                  {rule_GB6},
		VClass:                  {rule_GB7},
		LVClass:                 {rule_GB7},
		LVTClass:                {rule_GB8},
		TClass:                  {rule_GB8},
		ExtendClass:             {rule_GB9},
		ZWJClass:                {rule_GB9},
		SpacingMarkClass:        {rule_GB9a},
		PrependClass:            {rule_GB9b},
		emojiPictographic:       {rule_GB11},
		Regional_IndicatorClass: {rule_GB12},
	}
	gb.blocked = make(map[GraphemeClass]bool)
	return gb
}

// We introduce an offest for Emoji code-point classes
// to be able to tell them apart from grapheme classes.
const emojiPictographic GraphemeClass = ZWJClass + 1

// CodePointClassFor returns the grapheme code-point class for a rune (= code-point).
// (Interface uax.UnicodeBreaker)
func (gb *Breaker) CodePointClassFor(r rune) int {
	c := ClassForRune(r)
	if c == Any {
		if unicode.Is(emoji.Extended_Pictographic, r) {
			return int(emojiPictographic)
		}
	}
	return int(c)
}

// StartRulesFor starts all recognizers where the starting symbol is rune r.
// r is of code-point-class cpClass.
// (Interface uax.UnicodeBreaker)
//
// TODO merge this with ProceedWithRune(), it is unnecessary
func (gb *Breaker) StartRulesFor(r rune, cpClass int) {
	c := GraphemeClass(cpClass)
	if !gb.blocked[c] {
		if rules := gb.rules[c]; len(rules) > 0 {
			tracing.P("class", c).Debugf("starting %d rule(s)", c)
			for _, rule := range rules {
				rec := uax.NewPooledRecognizer(cpClass, rule)
				rec.UserData = gb
				gb.publisher.SubscribeMe(rec)
			}
		}
	}
}

// Helper: do not start any recognizers for this grapheme class, until
// unblocked again.
func (gb *Breaker) block(c GraphemeClass) {
	gb.blocked[c] = true
}

// Helper: stop blocking new recognizers for this grapheme class.
func (gb *Breaker) unblock(c GraphemeClass) {
	gb.blocked[c] = false
}

// ProceedWithRune is a signal to a Breaker:
// A new code-point has been read and this breaker receives a message to consume it.
// (Interface uax.UnicodeBreaker)
func (gb *Breaker) ProceedWithRune(r rune, cpClass int) {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("proceeding with rune %+q", r)
	gb.longestMatch, gb.penalties = gb.publisher.PublishRuneEvent(r, int(c))
	tracing.P("class", c).Debugf("...done with |match|=%d and %v", gb.longestMatch, gb.penalties)
	/*
		if c == Any { // rule GB999
			if len(gb.penalties) > 1 {
				gb.penalties[1] = uax.AddPenalties(gb.penalties[1], penaltyForAny)
			} else if len(gb.penalties) > 0 {
				gb.penalties = append(gb.penalties, penaltyForAny)
			} else {
				gb.penalties = penaltyForAnyAsSlice
			}
		}
	*/
	setPenalty1(gb, penalty999) //gb.penalties[1] = penalty999, if empty
}

// LongestActiveMatch collects information from
// all active recognizers about current match length
// and return the longest one for all still active recognizers.
// (Interface uax.UnicodeBreaker)
func (gb *Breaker) LongestActiveMatch() int {
	// We return a value of at least 1, as explained above.
	return max(1, gb.longestMatch)
	//return gb.longestMatch
}

// Penalties gets all active penalties for all active recognizers combined.
// Index 0 belongs to the most recently read rune, i.e., represents
// the penalty for breaking after it.
// (Interface uax.UnicodeBreaker)
func (gb *Breaker) Penalties() []int {
	return gb.penalties
}

// --- Rules ------------------------------------------------------------

// GlueBREAK, JOIN and BANG set default penalty values.
const (
	GlueBREAK  int = -500
	GlueJOIN   int = 10000
	GlueBANG   int = -20000
	penalty999 int = -10
)

// This is the break penalty for rule Any ÷ Any
const penaltyForAny = GlueBREAK

var penaltyForAnyAsSlice = []int{0, penaltyForAny}

// unnecessary ?!
/*
func rule_GB2(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	return uax.DoAccept(rec, 0, GlueBREAK)
}
*/

func rule_NewLine(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("fire rule NewLine")
	if c == LFClass {
		return uax.DoAccept(rec, GlueBANG, GlueBANG)
	} else if c == CRClass {
		rec.MatchLen++
		return rule_CRLF
	}
	return uax.DoAbort(rec)
}

func rule_CRLF(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("fire rule 05_CRLF")
	if c == LFClass {
		return uax.DoAccept(rec, GlueBANG, 3*GlueJOIN) // accept CR+LF
	}
	return uax.DoAccept(rec, 0, GlueBANG, GlueBANG) // accept CR
}

func rule_Control(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("fire rule Control")
	return uax.DoAccept(rec, GlueBANG, GlueBANG)
}

func rule_GB6(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//c := GraphemeClass(cpClass)
	rec.MatchLen++
	return rule_GB6_L_V_LV_LVT
}

func rule_GB6_L_V_LV_LVT(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	if c == LClass || c == VClass || c == LVClass || c == LVTClass {
		return uax.DoAccept(rec, 0, GlueJOIN)
	}
	return uax.DoAbort(rec)
}

func rule_GB7(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	//c := GraphemeClass(cpClass)
	rec.MatchLen++
	return rule_GB7_V_T
}

func rule_GB7_V_T(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	if c == VClass || c == TClass {
		return uax.DoAccept(rec, 0, GlueJOIN)
	}
	return uax.DoAbort(rec)
}

func rule_GB8(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("start rule GB8 LVT|T x T")
	rec.MatchLen++
	return rule_GB8_T
}

func rule_GB8_T(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("accept rule GB8 T")
	if c == TClass {
		return uax.DoAccept(rec, 0, GlueJOIN)
	}
	return uax.DoAbort(rec)
}

func rule_GB9(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("fire rule ZWJ|Extend")
	return uax.DoAccept(rec, 0, GlueJOIN)
}

func rule_GB9a(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("fire rule SpacingMark")
	return uax.DoAccept(rec, 0, GlueJOIN)
}

func rule_GB9b(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	tracing.P("class", c).Debugf("fire rule Preprend")
	return uax.DoAccept(rec, GlueJOIN)
}

func rule_GB11(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	tracing.P("class", cpClass).Debugf("fire rule Emoji Pictographic")
	return rule_GB11Cont
}

func rule_GB11Cont(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	if cpClass == int(ZWJClass) {
		rec.MatchLen++
		return rule_GB11Finish
	} else if cpClass == int(ExtendClass) {
		rec.MatchLen++
		return rule_GB11Cont
	}
	return uax.DoAbort(rec)
}

func rule_GB11Finish(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	if cpClass == int(emojiPictographic) {
		return uax.DoAccept(rec, 0, GlueJOIN)
	}
	return uax.DoAbort(rec)
}

func rule_GB12(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	tracing.P("class", cpClass).Debugf("fire rule RI")
	gb := rec.UserData.(*Breaker)
	gb.block(Regional_IndicatorClass)
	return rule_GB12Cont
}

func rule_GB12Cont(rec *uax.Recognizer, r rune, cpClass int) uax.NfaStateFn {
	c := GraphemeClass(cpClass)
	gb := rec.UserData.(*Breaker)
	gb.unblock(Regional_IndicatorClass)
	if c == Regional_IndicatorClass {
		return uax.DoAccept(rec, 0, GlueJOIN)
	}
	return uax.DoAbort(rec)
}

// ---------------------------------------------------------------------------

func capw(w int) int {
	if w < 0 {
		return w
	}
	if w > 5 {
		return 5
	}
	return w
}

func setPenalty1(gb *Breaker, p int) {
	if len(gb.penalties) == 0 {
		gb.penalties = append(gb.penalties, 0)
		gb.penalties = append(gb.penalties, p)
	} else if len(gb.penalties) == 1 {
		gb.penalties = append(gb.penalties, p)
	} else if gb.penalties[1] == 0 {
		gb.penalties[1] = p
	}
}
