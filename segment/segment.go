/*
Package segment is about Unicode text segmenting.

Under active development; use at your own risk

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
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


Typical Usage

Segmenter provides an interface similar to bufio.Scanner for reading data
such as a file of Unicode text.
Similar to Scanner's Scan() function, successive calls to a segmenter's
Next() method will step through the 'segments' of a file.
Clients are able to get runes of the segment by calling Bytes() or Text().
Unlike Scanner, segmenters are calculating a 'penalty' for breaking
at this segment. Penalties are numeric values and reflect costs, where
negative values are to be interpreted as negative costs, i.e. merits.

Clients instantiate a UnicodeBreaker object and use it as the
breaking engine for a segmenter. Multiple breaking engines may be
supplied (where the first one is called the primary breaker and any
following breaker is a secondary breaker).

  breaker1 := ...
  breaker2 := ...
  segmenter := unicode.NewSegmenter(breaker1, breaker2)
  segmenter.Init(...)
  for segmenter.Next() {
    // do something with segmenter.Text() or segmenter.Bytes()
  }

An example for an UnicodeBreaker is "uax29.WordBreak", a breaker
implementing the UAX#29 word breaking algorithm.

How it works

The segmenter uses a double-ended queue to collect runes and the
breaking opportunities between them. The front of the queue keeps
adding new runes, while at the end of the queue we withdraw segments
as soon as they are available.
(see https://github.com/npillmayer/gotype/wiki).

For every rune r read, the segmenter will fire up all the rules which
start with r. It is not uncommon that the lifetime of a lot of rules
overlap and all those rules are adding breaking information. */
package segment

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/npillmayer/schuko/gtrace"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax"
)

// CT traces to the core-tracer.
func CT() tracing.Trace {
	return gtrace.CoreTracer
}

// A Segmenter receives a sequence of code-points from an io.RuneReader and
// segments it into smaller parts, called segments.
//
// The specification of a segment is defined by a breaker function of type
// UnicodeBreaker; the default UnicodeBreaker breaks the input into words,
// using whitespace as boundaries. For more sophisticated breakers see
// sub-packages of package uax.
type Segmenter struct {
	deque         *deque               // where we collect runes and penalties
	reader        io.RuneReader        // where we get the next runes from
	breakers      []uax.UnicodeBreaker // our work horses
	activeSegment []byte               // the most recent segment to build
	buffer        *bytes.Buffer        // wrapper around activeSegment
	lastPenalties [2]int               // penalties at last break opportunity
	maxSegmentLen int                  // maximum length allowed for segments
	pos           int64                // current position in text
	breakOnZero   [2]bool              // treat zero value as a valid breakpoint?
	//primaryAgg, secondaryAgg   uax.PenaltyAggregator
	//primarySeed, secondarySeed int
	longestActiveMatch         int
	positionOfBreakOpportunity int
	err                        error
	atEOF                      bool
	inUse                      bool // Next() has been called; buffer is in use.
}

// MaxSegmentSize is the maximum size used to buffer a segment
// unless the user provides an explicit buffer with Segmenter.Buffer().
const MaxSegmentSize = 64 * 1024
const startBufSize = 4096 // Size of initial allocation for buffer.

// ErrTooLong flags a buffer overflow.
// ErrNotInitialized is returned if a segmenters Next-function is called without
// first setting an input source.
var (
	ErrTooLong        = errors.New("UAX segmenter: segment too long for buffer")
	ErrNotInitialized = errors.New("UAX segmenter not initialized; must call Init(...) first")
)

// NewSegmenter creates a new Segmenter by providing breaking logic (UnicodeBreaker).
// Clients may provide more than one UnicodeBreaker. Specifying no
// UnicodeBreaker results in getting a SimpleWordBreaker, which will
// break on whitespace (see SimpleWordBreaker in this package).
//
// Before using newly created segmenters, clients will have to call Init(...)
// on them, i.e. initialize them for a rune reader.
func NewSegmenter(breakers ...uax.UnicodeBreaker) *Segmenter {
	s := &Segmenter{}
	if len(breakers) == 0 {
		breakers = []uax.UnicodeBreaker{NewSimpleWordBreaker()}
	}
	s.breakers = breakers
	// s.primaryAgg = uax.AddPenalties
	// s.secondaryAgg = uax.AddPenalties
	return s
}

// Init initializes a Segmenter with an io.RuneReader to read from.
// s is either a newly created segmenter to be initialized, or we may
// re-initializes a segmenter already in use.
func (s *Segmenter) Init(reader io.RuneReader) {
	if reader == nil {
		reader = strings.NewReader("")
	}
	s.reader = reader
	if s.deque == nil {
		s.deque = &deque{} // Q of atoms
		s.buffer = bytes.NewBuffer(make([]byte, 0, startBufSize))
		s.maxSegmentLen = MaxSegmentSize
	} else {
		s.deque.Clear()
		s.longestActiveMatch = 0
		s.atEOF = false
		s.buffer.Reset()
		s.inUse = false
		s.lastPenalties[0], s.lastPenalties[1] = 0, 0
		s.pos = 0
	}
	s.positionOfBreakOpportunity = -1
}

// Buffer sets the initial buffer to use when scanning and the maximum size of
// buffer that may be allocated during segmenting.
// The maximum segment size is the larger of max and cap(buf).
// If max <= cap(buf), Next() will use this buffer only and do no allocation.
//
// By default, Segmenter uses an internal buffer and sets the maximum token size
// to MaxSegmentSize.
//
// Buffer panics if it is called after scanning has started. Clients will have
// to call Init(...) again to permit re-setting the buffer.
func (s *Segmenter) Buffer(buf []byte, max int) {
	if s.inUse {
		panic("segment.Buffer: buffer already in use; cannot be re-set")
	}
	s.buffer = bytes.NewBuffer(buf)
	s.maxSegmentLen = max
}

// Err returns the first non-EOF error that was encountered by the
// Segmenter.
func (s *Segmenter) Err() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}

func (s *Segmenter) BreakOnZero(forP1, forP2 bool) {
	s.breakOnZero[0] = forP1
	s.breakOnZero[1] = forP2
}

// Set the null-value for penalties. Penalties equal to this value will
// be treated as if no penalty occured (possibly resulting in the
// suppression of a break opportunity).
//
// There is one null-value for each UnicodeBreaker. The segmenter issues
// a break whenever one of the UnicodeBreakers signals a non-null penalty.
// The default null-value function treats any penalty >= 1000 as a null,
// i.e. suppresses the break opportunity.
//
// bInx is the position 0..n-1 of the UnicodeBreaker as provided during
// construction of the segmenter.
// The call to SetNullPenalty panics if bInx is out of range.
/*
func (s *Segmenter) SetNullPenalty(bInx int, isNull func(int) bool) {
	if bInx < 0 || bInx >= len(s.breakers) {
		panic("segment.SetNullPenalty: Index of UnicodeBreaker out of range!")
	}
	if isNull == nil {
		s.nullPenalty[bInx] = tooBad
	} else {
		s.nullPenalty[bInx] = isNull
	}
}
*/

// Penalties >= InfinitePenalty are considered too bad for being a break opportunity.
func isPossibleBreak(p int, breakOnZero bool) bool {
	if p >= uax.InfinitePenalty {
		return false
	}
	if !breakOnZero && p == 0 {
		return false
	}
	return true
}

// Next gets the next segment, together with the accumulated penalty for this break.
//
// Next() advances the Segmenter to the next segment, which will then be available
// through the Bytes() or Text() method. It returns false when the segmenting
// stops, either by reaching the end of the input or an error.
// After Next() returns false, the Err() method will return any error
// that occurred during scanning, except for io.EOF.
// For the latter case Err() will return nil.
//
func (s *Segmenter) Next() bool {
	return s.next(math.MaxInt64)
}

// BoundedNext gets the next segment, together with the accumulated penalty for this break.
//
// BoundedNext() advances the Segmenter to the next segment, which will then be available
// through the Bytes() or Text() method. It returns false when the segmenting
// stops, either by reaching the end of the input, reaching the bound, or if an
// error occurs.
// After BoundedNext() returns false, the Err() method will return any error
// that occurred during scanning, except for io.EOF.
// For the latter case Err() will return nil.
//
// See also method `Next`.
//
func (s *Segmenter) BoundedNext(bound int64) bool {
	if s.pos >= bound {
		return false
	}
	return s.next(bound)
}

// next advances the input pointer until a possible break point has been
// found or alternatively the bound has been reached while trying.
func (s *Segmenter) next(bound int64) bool {
	if s.reader == nil {
		s.setErr(ErrNotInitialized)
	}
	s.inUse = true
	if !s.atEOF {
		err := s.readEnoughInput(bound)
		if err != nil && err != io.EOF {
			s.setErr(err)
			s.activeSegment = nil
			return false
		}
	}
	CT().Debugf("----- have read enough input ----")
	if s.positionOfBreakOpportunity < 0 { // didn't find a break opportunity
		if false && s.pos >= bound { // TODO no opportunity, but reached bound => return segment
			l := s.copySegment(s.buffer)
			s.activeSegment = s.buffer.Bytes()
			CT().P("length", strconv.Itoa(l)).Debugf("BNext()|= \"%v\"", string(s.activeSegment))
			return true
		}
		// otherwise do not return anything
		s.activeSegment = nil
		return false
	}
	l := s.getFrontSegment(s.buffer)
	s.activeSegment = s.buffer.Bytes()
	CT().P("length", strconv.Itoa(l)).Debugf("Next() = \"%v\"", string(s.activeSegment))
	// if s.pos >= bound {
	// 	CT().Debugf("=================================")
	// 	return false
	// }
	return true
}

// Bytes returns the most recent token generated by a call to Next().
// The underlying array may point to data that will be overwritten by a
// subsequent call to Next(). No allocation is performed.
func (s *Segmenter) Bytes() []byte {
	return s.activeSegment
}

// Text returns the most recent segment generated by a call to Next()
// as a newly allocated string holding its bytes.
func (s *Segmenter) Text() string {
	return string(s.activeSegment)
}

// Penalties returns the last penalties a segmenter calculated.
// Two penalties are returned. The first one is the penalty returned from the
// primary breaker, the second one is the aggregate of all penalties of all the
// secondary breakers (if any).
func (s *Segmenter) Penalties() (int, int) {
	return s.lastPenalties[0], s.lastPenalties[1]
}

// setErr() records the first error encountered.
func (s *Segmenter) setErr(err error) {
	if s.err == nil || s.err == io.EOF {
		s.err = err
	}
}

// SetPenaltyAggregator sets an aggregate function for penalties from the primary
// breaker.
// Default is uax.AddPenalties. Not all aggregators may be monoids; for
// aggregators which are semi-groups (i.e., have not neutral element), a seed
// is required to give a starting
// point for aggregation.
// func (s *Segmenter) SetPenaltyAggregator(pa uax.PenaltyAggregator, seed int) {
// 	if pa != nil {
// 		s.primaryAgg = pa
// 		s.primarySeed = seed
// 	}
// }

// SetSecondaryPenaltyAggregator sets an aggregate function for penalties
// from all the secondary breaks.
// Default is uax.AddPenalties. Not all aggregators may be monoids; for
// aggregators which are semi-groups (i.e., have not neutral element), a seed
// is required to give a starting
// point for aggregation.
// func (s *Segmenter) SetSecondaryPenaltyAggregator(pa uax.PenaltyAggregator, seed int) {
// 	if pa != nil {
// 		s.secondaryAgg = pa
// 		s.secondarySeed = seed
// 	}
// }

func (s *Segmenter) readRune() error {
	if s.atEOF {
		return io.EOF
	}
	r, sz, err := s.reader.ReadRune()
	s.pos += int64(sz)
	CT().P("rune", fmt.Sprintf("%#U", r)).Debugf("--------------------------------------")
	if err == nil {
		//s.deque.PushBack(r, s.primarySeed, s.secondarySeed)
		s.deque.PushBack(r, 0, 0)
	} else if err == io.EOF {
		s.deque.PushBack(eotAtom.r, eotAtom.penalty0, eotAtom.penalty1)
		s.atEOF = true
		err = nil
	} else {
		CT().P("rune", fmt.Sprintf("%#U", r)).Errorf("ReadRune() error: %s", err)
		s.atEOF = true
	}
	return err
}

var errBoundReached = errors.New("bound reached")

func (s *Segmenter) readEnoughInput(bound int64) (err error) {
	for s.positionOfBreakOpportunity < 0 && s.pos < bound {
		l := s.deque.Len()
		if s.pos-int64(s.longestActiveMatch) >= bound {
			CT().Errorf("===> BOUND REACHED")
		}
		err = s.readRune()
		if err != nil {
			break
		}
		if s.deque.Len() == l {
			panic("segmenter: code-point deque did not grow") // TODO remove this after extensive testing
		}
		from := max(0, l-1-s.longestActiveMatch) // current longest match limit, now old
		// TODO if from >= bound: exit loop
		// avoid to read rune on re-enter of this loop
		//
		l = s.deque.Len()
		s.longestActiveMatch = 0
		r, _, _ := s.deque.Back()
		for _, breaker := range s.breakers {
			cpClass := breaker.CodePointClassFor(r)
			breaker.StartRulesFor(r, cpClass)
			breaker.ProceedWithRune(r, cpClass)
			if breaker.LongestActiveMatch() > s.longestActiveMatch {
				s.longestActiveMatch = breaker.LongestActiveMatch()
			}
			s.insertPenalties(s.inxForBreaker(breaker), breaker.Penalties())
		}
		s.positionOfBreakOpportunity = s.findBreakOpportunity(from, l-1-s.longestActiveMatch)
		//s.positionOfBreakOpportunity = s.findBreakOpportunity(from, l-s.longestActiveMatch)
		CT().Debugf("segmenter: breakpos = %d, active match = %d", s.positionOfBreakOpportunity, s.longestActiveMatch)
		s.printQ()
	}
	return err
}

func (s *Segmenter) findBreakOpportunity(from int, to int) int {
	pos := -1
	CT().Debugf("segmenter: searching for break opportunity from %d to %d: ", from, to-1)
	for i := 0; i < to; i++ {
		j, p0, p1 := s.deque.At(i)
		CT().Debugf("segmenter: penalties[%#U] = %d|%d", j, p0, p1)
		if isPossibleBreak(p0, s.breakOnZero[0]) || (len(s.breakers) > 1 && isPossibleBreak(p1, s.breakOnZero[1])) {
			pos = i
			break
		}
	}
	CT().Debugf("segmenter: break opportunity at %d", pos)
	return pos
}

// find out if the UnicodeBreaker b is the primary breaker
func (s *Segmenter) inxForBreaker(b uax.UnicodeBreaker) int {
	if b == s.breakers[0] {
		return 0
	}
	return 1
}

func (s *Segmenter) insertPenalties(selector int, penalties []int) {
	l := s.deque.Len()
	if len(penalties) > l {
		penalties = penalties[0:l] // drop excessive penalties
	}
	for i, p := range penalties {
		r, total0, total1 := s.deque.At(l - 1 - i)
		if selector == 0 {
			// total0 = s.primaryAgg(total0, p)
			//total0 += p
			total0 = bounded(total0 + p)
		} else {
			// total1 = s.secondaryAgg(total1, p)
			//total1 += p
			total1 = bounded(total1 + p)
		}
		s.deque.SetAt(l-1-i, r, total0, total1)
	}
}

func (s *Segmenter) getFrontSegment(buf *bytes.Buffer) int {
	seglen := 0
	s.lastPenalties[0] = 0
	s.lastPenalties[1] = 0
	buf.Reset()
	l := min(s.deque.Len()-1, s.positionOfBreakOpportunity)
	CT().Debugf("cutting front segment of length 0..%d", l)
	if l > buf.Len() {
		if l > s.maxSegmentLen {
			s.setErr(ErrTooLong)
			return 0
		}
		newSize := max(buf.Len()+startBufSize, l+1)
		if newSize > s.maxSegmentLen {
			newSize = s.maxSegmentLen
		}
		buf.Grow(newSize)
	}
	cnt := 0
	for i := 0; i <= l; i++ {
		r, p0, p1 := s.deque.PopFront()
		written, _ := buf.WriteRune(r)
		seglen += written
		cnt++
		s.lastPenalties[0] = p0
		s.lastPenalties[1] = p1
	}
	CT().Debugf("front segment is of length %d/%d", seglen, cnt)
	s.positionOfBreakOpportunity = s.findBreakOpportunity(0, s.deque.Len()-1-s.longestActiveMatch)
	s.printQ()
	return seglen
}

func (s *Segmenter) copySegment(buf *bytes.Buffer) int {
	buf.Reset()
	l := s.deque.Len() - 1
	if l > buf.Len() {
		if l > s.maxSegmentLen {
			s.setErr(ErrTooLong)
			return 0
		}
		newSize := max(buf.Len()+startBufSize, l+1)
		if newSize > s.maxSegmentLen {
			newSize = s.maxSegmentLen
		}
		buf.Grow(newSize)
	}
	seglen, cnt := 0, 0
	for i := 0; i <= l; i++ {
		r, p0, p1 := s.deque.At(i)
		written, _ := buf.WriteRune(r)
		seglen += written
		cnt++
		s.lastPenalties[0] = p0
		s.lastPenalties[1] = p1
	}
	return seglen
}

// ----------------------------------------------------------------------

// Debugging helper. Print the content of the current queue to the debug log.
func (s *Segmenter) printQ() {
	if CT().GetTraceLevel() <= tracing.LevelDebug {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Q #%d: ", s.deque.Len()))
	for i := 0; i < s.deque.Len(); i++ {
		var a atom
		a.r, a.penalty0, a.penalty1 = s.deque.At(i)
		sb.WriteString(fmt.Sprintf(" <- %s", a.String()))
	}
	sb.WriteString(" .")
	CT().Debugf(sb.String())
}

// --- Helpers ----------------------------------------------------------

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func bounded(p int) int {
	if p > uax.InfinitePenalty {
		p = uax.InfinitePenalty
	} else if p < uax.InfiniteMerits {
		p = uax.InfiniteMerits
	}
	return p
}
