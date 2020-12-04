package uax

import (
	"context"
	"fmt"

	pool "github.com/jolestar/go-commons-pool"
)

// UnicodeBreaker represents a logic to split up
// Unicode sequences into smaller parts. They are used by Segmenters
// to supply breaking logic.
type UnicodeBreaker interface {
	CodePointClassFor(rune) int
	StartRulesFor(rune, int)
	ProceedWithRune(rune, int)
	LongestActiveMatch() int
	Penalties() []int
}

// NfaStateFn represents a state in a non-deterministic finite automata.
// Functions of type NfaStateFn try to match a rune (Unicode code-point).
// The caller may provide a third argument, which should be a rune class.
// Rune (code-point) classes are described in various Unicode standards
// and annexes. One such annex, UAX#29, describes classes to help
// splitting up text into graphemes or words. An example class may be
// a class of western language alphabetic characters "AL", of which runes
// 'A' and 'é' would be part of.
//
// The first argument is a Recognizer (see the definition of
// type Recognizer in this package), which carries this state function.
//
// NfaStateFn – after matching a rune – must return another NfaStateFn,
// which will then in turn be called to process the next rune. The process
// of matching a string will stop as soon as a NfaStateFn returns nil.
type NfaStateFn func(*Recognizer, rune, int) NfaStateFn

// A Recognizer represents an automata to recognize sequences of runes
// (i.e. Unicode code-points). Its main functionality is performed by
// an embedded NfaStateFn. The first NfaStateFn to use is provided with
// the constructor.
//
// Recognizer's state functions must be careful to increment MatchLen
// with each matched rune. Failing to do so may result in incorrect splits
// of text.
//
// Semantics of Expect and UserData are up to the client and not used by
// the default mechanism.
//
// It is not mandatory to use Recognizers for segmenting text. The type is
// provided for easier implementation of types implementing UnicodeBreaker.
// Recognizers implement interface RuneSubscriber and UnicodeBreakers will
// use a UnicodePublisher to interact with them.
type Recognizer struct {
	Expect    int         // next code-point to expect; semantics are up to the client
	MatchLen  int         // length of active match
	UserData  interface{} // clients may need to store additional information
	penalties []int       // penalties to return, used internally in DoAccept()
	nextStep  NfaStateFn  // next step of a DFA
}

// NewRecognizer creates a new Recognizer.
// This is rarely used, as clients rather should call NewPooledRecognizer().
//
// see NewPooledRecognizer.
func NewRecognizer(codePointClass int, next NfaStateFn) *Recognizer {
	rec := &Recognizer{}
	rec.Expect = codePointClass
	rec.nextStep = next
	return rec
}

// Recognizers are short-lived objects. To avoid multiple allocation of
// small objects we will pool them.
type recognizerPool struct {
	opool *pool.ObjectPool
	ctx   context.Context
}

var globalRecognizerPool *recognizerPool

func init() {
	globalRecognizerPool = &recognizerPool{}
	factory := pool.NewPooledObjectFactorySimple(
		func(context.Context) (interface{}, error) {
			rec := &Recognizer{}
			return rec, nil
		})
	globalRecognizerPool.ctx = context.Background()
	config := pool.NewDefaultPoolConfig()
	//config.LIFO = false
	config.MaxTotal = -1 // infinity
	config.BlockWhenExhausted = false
	globalRecognizerPool.opool = pool.NewObjectPool(globalRecognizerPool.ctx, factory, config)
}

// NewPooledRecognizer returns a new Recognizer, pre-filled with an expected code-point class
// and a state function. The Recognizer is pooled for efficiency.
func NewPooledRecognizer(cpClass int, stateFn NfaStateFn) *Recognizer {
	o, _ := globalRecognizerPool.opool.BorrowObject(globalRecognizerPool.ctx)
	rec := o.(*Recognizer)
	rec.Expect = cpClass
	rec.nextStep = stateFn
	return rec
}

// Clears the Recognizer and puts it back into the pool.
func (rec *Recognizer) releaseIntoPool() {
	rec.penalties = nil
	rec.Expect = 0
	rec.MatchLen = 0
	rec.nextStep = nil
	_ = globalRecognizerPool.opool.ReturnObject(globalRecognizerPool.ctx, rec)
}

// Simple stringer for debugging purposes.
func (rec *Recognizer) String() string {
	if rec == nil {
		return "[nil rule]"
	}
	return fmt.Sprintf("[%d -> done=%v]", rec.Expect, rec.Done())
}

// Unsubscribed signals to
// a Recognizer that it has been unsubscribed from a RunePublisher;
// usually after the Recognizer's NfaStateFn has returned nil.
//
// Interface RuneSubscriber
func (rec *Recognizer) Unsubscribed() {
	rec.releaseIntoPool()
}

// Done is used by a Recognizer that it is done matching runes.
// If MatchLength() > 0 is has been accepting a sequence of runes,
// otherwise it has aborted to further try a match.
//
// Interface RuneSubscriber
func (rec *Recognizer) Done() bool {
	return rec.nextStep == nil
}

// MatchLength is part of interface RuneSubscriber.
func (rec *Recognizer) MatchLength() int {
	return rec.MatchLen
}

// RuneEvent is part  of interface RuneSubscriber.
func (rec *Recognizer) RuneEvent(r rune, codePointClass int) []int {
	//fmt.Printf("received rune event: %+q / %d\n", r, codePointClass)
	var penalties []int
	if rec.nextStep != nil {
		//CT.Infof("  calling func = %v", rec.nextStep)
		rec.nextStep = rec.nextStep(rec, r, codePointClass)
	} else {
		//CT.Info("  not calling func = nil")
	}
	if rec.Done() && rec.MatchLen > 0 { // accepted a match
		penalties = rec.penalties
	}
	//CT.Infof("    subscriber:      penalites = %v, done = %v, match len = %d", penalties, rec.Done(), rec.MatchLength())
	return penalties
}

// --- Standard Recognizer Rules ----------------------------------------

// DoAbort returns a state function which signals abort.
func DoAbort(rec *Recognizer) NfaStateFn {
	rec.MatchLen = 0
	return nil
}

// DoAccept returns a state function which signals accept, together with break
// penalties for matches runes (in reverse sequence).
func DoAccept(rec *Recognizer, penalties ...int) NfaStateFn {
	rec.MatchLen++
	rec.penalties = penalties
	CT().Debugf("ACCEPT with %v", rec.penalties)
	return nil
}

// --- Rune Publishing and Subscription ---------------------------------

// A RuneSubscriber is a receiver of rune events, i.e. messages to
// process a new code-point (rune). If they can match the rune, they
// will expect further runes, otherwise they abort. To they are finished,
// either by accepting or rejecting input, they set Done() to true.
// A successful acceptance of input is signalled by Done()==true and
// MatchLength()>0.
type RuneSubscriber interface {
	RuneEvent(r rune, codePointClass int) []int // receive a new code-point
	MatchLength() int                           // length (in # of code-point) of the match up to now
	Done() bool                                 // is this subscriber done?
	Unsubscribed()                              // this subscriber has been unsubscribed
}

// A RunePublisher notifies subscribers with rune events: a new rune has been read
// and the subscriber – usually a recognizer rule – has to react to it.
//
// UnicodeBreakers are not required to use the RunePublisher/RuneSubscriber
// pattern, but it is convenient to stick to it. UnicodeBreakers often
// rely on sets of rules, which are tested interleavingly. To releave
// UnicodeBreakers from managing rune-distribution to all the rules, it
// may be advantageous hold a RunePublisher within a UnicodeBreaker and
// let all rules implement the RuneSubscriber interface.
type RunePublisher interface {
	SubscribeMe(RuneSubscriber) RunePublisher // subscribe an additional rune subscriber
	PublishRuneEvent(r rune, codePointClass int) (longestDistance int, penalties []int)
	SetPenaltyAggregator(pa PenaltyAggregator) // function to aggregate break penalties
}

// NewRunePublisher creates a new default RunePublisher.
func NewRunePublisher() *DefaultRunePublisher {
	rpub := &DefaultRunePublisher{}
	rpub.aggregate = AddPenalties
	return rpub
}

// PublishRuneEvent triggers a rune event notification to all subscribers. Rune events
// include the rune (code-point) and an optional code-point class for
// the rune.
//
// Return values are: the longest active match and a slice of penalties.
// These values usually are collected from the RuneSubscribers.
// Penalties will be overwritten by the next call to PublishRuneEvent().
// Clients will have to make a copy if they want to preserve penalty
// values.
//
// Interface RunePublisher
func (rpub *DefaultRunePublisher) PublishRuneEvent(r rune, codePointClass int) (int, []int) {
	longest := 0
	if rpub.penaltiesTotal == nil {
		rpub.penaltiesTotal = make([]int, 1024)
	}
	//CT.Infof("pre-publish(): total penalites = %v", rpub.penaltiesTotal)
	rpub.penaltiesTotal = rpub.penaltiesTotal[:0]
	//CT.Infof("pre-publish(): total penalites = %v", rpub.penaltiesTotal)
	// pre-condition: no subscriber is Done()
	for i := rpub.Len() - 1; i >= 0; i-- {
		subscr := rpub.at(i)
		penalties := subscr.RuneEvent(r, codePointClass)
		//CT.Infof("    publish():       penalites = %v", penalties)
		for j, p := range penalties { // aggregate all penalties
			if j >= len(rpub.penaltiesTotal) {
				rpub.penaltiesTotal = append(rpub.penaltiesTotal, p)
			} else {
				rpub.penaltiesTotal[j] = rpub.aggregate(rpub.penaltiesTotal[j], p)
			}
		}
		//CT.Infof("    publish(): total penalites = %v", rpub.penaltiesTotal)
		if !subscr.Done() { // compare against longest active match
			if d := subscr.MatchLength(); d > longest {
				longest = d
			}
		}
		rpub.Fix(i) // re-order heap if subscr.Done()
	}
	//CT.Infof("pre-publish(): total penalites = %v", rpub.penaltiesTotal)
	// now unsubscribe all done subscribers
	for subscr := rpub.PopDone(); subscr != nil; subscr = rpub.PopDone() {
		subscr.Unsubscribed()
	}
	return longest, rpub.penaltiesTotal
}

// PenaltyAggregator is a
// function type for methods of penalty-aggregation. Aggregates all the
// break penalties each a break-point to a single penalty value at that point.
type PenaltyAggregator func(int, int) int

// SetPenaltyAggregator sets a PenaltyAggregator for a rune publisher.
// A PenaltyAggregator aggregates all the
// break penalties each a break-point to a single penalty value at that point.
//
// Part of interface RunePublisher.
func (rpub *DefaultRunePublisher) SetPenaltyAggregator(pa PenaltyAggregator) {
	if pa == nil {
		rpub.aggregate = AddPenalties
	} else {
		rpub.aggregate = pa
	}
}

// AddPenalties is the default function to aggregate break-penalties.
// Simply adds up all penalties at each break position, respectively.
func AddPenalties(total int, p int) int {
	return total + p
}

// MaxPenalties is an alternative function to aggregate break-penalties.
// Returns maximum of all penalties at each break position.
func MaxPenalties(total int, p int) int {
	return max(total, p)
}

// SubscribeMe lets a client subscribe to a RunePublisher.
//
// Part of interface RunePublisher.
func (rpub *DefaultRunePublisher) SubscribeMe(rsub RuneSubscriber) RunePublisher {
	if rpub.aggregate == nil { // this is necessary as we allow uninitialzed DefaultRunePublishers
		rpub.aggregate = AddPenalties
	}
	rpub.Push(rsub)
	return rpub
}

// ----------------------------------------------------------------------

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
