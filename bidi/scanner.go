package bidi

// Notes:
// - Input reader will in most cases be a cord reader
// - LBRACKC and RBRACKC are no longer used -> TODO remove them
//   this may help further reducing the size of the trie
// - TODO set MAX = NI, this may help further reducing the size of the trie

import (
	"bufio"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/bidi"
)

// Scanner implements the scanner.Tokenizer interface.
// It will read runs of text as a unit, as long as all runes therein have the
// same Bidi_Class.
type bidiScanner struct {
	//done        bool           // at EOF?
	mode        uint8          // scanner modes, set by scanner options
	bidiclz     bidi.Class     // the current bidi class
	lookahead   scrap          // lookahead has been read but not sent to parser
	pos         charpos        // position in input string
	ahead       charpos        // position ahead of current scrap
	strtyps     strongTypes    // positions of previously occured strong types
	runeScanner *bufio.Scanner // we're using an embedded rune reader
	bd16stack   bracketStack   // bracket pair stack, rule BD16
	//laLen       charpos        // byte length of lookahead
	// lookahead   []byte         // lookahead rune
	// buffer      []byte         // character buffer for token lexeme
}

// BS16Max is the maximum stack depth for rule BS16 as defined in UAX#9.
const BS16Max = 63

// NewScanner creates a scanner for bidi formatting. It will read runs of text
// as a unit, as long as all runes therein have the same Bidi_Class.
//
// Clients will provide a Reader and zero or more scanner options. Runes will be
// read from the reader and possibly concatenated to chunks of text with
// identical BiDi class (see NextToken).
func newScanner(input io.Reader, opts ...Option) *bidiScanner {
	sc := &bidiScanner{}
	sc.runeScanner = bufio.NewScanner(input)
	sc.runeScanner.Split(bufio.ScanRunes)
	sc.bidiclz = bidi.LRI
	//sc.buffer = make([]byte, 0, buflen)
	//sc.lookahead = make([]byte, 0, 32)
	sc.bd16stack = make(bracketStack, 0, BS16Max+1)
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

func (sc *bidiScanner) nextRune() (rune, int, bidi.Class, bool) {
	ok := sc.runeScanner.Scan()
	if !ok {
		return 0, 0, ILLEGAL, false
	}
	b := sc.runeScanner.Bytes()
	r, length := utf8.DecodeRune(b)
	props, sz := bidi.LookupRune(r)
	if sz == 0 || sz != length {
		panic("bidi package differs in rune interpretation from scanner package")
	}
	bidiclz := props.Class()
	return r, length, bidiclz, true
}

func (sc *bidiScanner) makeScrap(r rune, pos, ahead charpos, length int) scrap {
	s := scrap{
		l:       pos,
		r:       pos + charpos(length),
		bidiclz: sc.bidic(r),
	}
	clz, openbrpos, isbr := sc.doBD16(r, ahead, s.bidiclz, true)
	if isbr {
		s.bidiclz = clz
		if openbrpos >= 0 {
			s.r = openbrpos // bracket scraps mis-use r for position of opening bracket
		}
	}
	return s
}

// Scan reads the next run of input text with identical bidi_class, returning a scrap for
// it. The scanner needs not return the input's lexeme, as it will not be processed by the
// parser. The parser operates on intervals of bidi clusters without caring about
// individual characters.
//
// Attention:
// The scanner should operate on one paragraph at a time, as required by UAX#9.
// It will manage internal counters that may overflow when scanning complete texts.
// As opposed to the generic scanner interface, which will handle character positions
// as uint64, the bidi scanner has certain internal limits which have to fit into
// uint16.
//
func (sc *bidiScanner) Scan(pipe chan<- scrap) {
	var pos, lapos charpos
	var current, lookahead scrap
	current.bidiclz = ILLEGAL
	for {
		r, length, bidiclz, ok := sc.read()
		if !ok {
			sc.post(lookahead, pipe)
			sc.stop(pipe)
			break
		}
		lookahead = makeScrap(r, sc.pos, sc.ahead, length)
		T().Debugf("bidi scanner read lookahead = %v", lookahead)
		if current.bidiclz == ILLEGAL || current.bidiclz != lookahead.bidiclz || isbracket(current.bidiclz) {
			if current.bidiclz != ILLEGAL {
				sc.post(lookahead, pipe) // put current on channel
			}
			current = lookahead
			pos = lapos
			lapos += lookahead.len()
		} else {
			collapse(current, lookahead, current.bidiclz) // append LA to current
		}
	}
}

func (sc *bidiScanner) post(s scrap, pipe chan<- scrap) {
	T().Debugf("bidi scanner sends current scrap: %v", s)
	pipe <- s
}

func (sc *bidiScanner) stop(pipe chan<- scrap) {
	T().Debugf("stopping bidi scanner, sending final scrap (stopper)")
	s := scrap{bidiclz: ILLEGAL}
	pipe <- s
	T().Infof("stopped bidi scanner")
}

/* func (sc *bidiScanner) XNextToken(expected []int) (int, interface{}, uint64, uint64) {
	sc.prepareNewRun()
	//T().Debugf("re-reading '%s'", string(sc.buffer))
	var isBracket bool
	var openpos int64
	for sc.runeScanner.Scan() {
		b := sc.runeScanner.Bytes()
		//T().Debugf("--------------")
		r, clz, sz := sc.bidic(b)
		//T().Debugf("next char '%s' has class %s", string(b), ClassString(clz))
		//clz, _, isBracket = sc.doBD16(r, sc.ahead, clz, false)
		clz, isBracket := sc.checkBD16(r, clz)
		if clz != sc.bidiclz || isBracket { // bidi classes get collected, but brackets dont't
			sc.lookahead = sc.lookahead[:0]           // truncate previous LA
			sc.lookahead = append(sc.lookahead, b...) // and replace by last read rune
			current := sc.bidiclz                     // tmp for returning current class
			sc.bidiclz = clz                          // change current class to class of LA
			sc.setStrongPos(current)                  // remember if current is a strong type
			T().Debugf("scanned Token '%s' as :%s", string(sc.buffer), ClassString(current))
			l := uint64(len(sc.buffer)) // length of current bidi cluster
			_, openpos, isBracket = sc.doBD16(r, sc.ahead, current, false)
			T().Debugf("is bracket = %v", isBracket)
			if isBracket {
				if sc.bidiclz == BRACKC {
					//_, openpos, _ = sc.doBD16(r, sc.ahead, clz, true)
					T().Debugf("position of opening bracket is %d", openpos)
					T().Debugf("sc.ahead is %d", sc.ahead)
					if current == BRACKC {
						// We have the closing bracket to a corresponding open one.
						// Brackets are always of length 1, so we'll misuse the length field
						// for some other information: the position of the openending bracket
						// if this is a closing bracket.
						l = uint64(openpos)
					}
				}
			}
			if current == bidi.AL {
				current = bidi.R // rule W3
			}
			return int(current), sc.strtyps, sc.pos, l
		}
		sc.buffer = append(sc.buffer, b...)
		sc.ahead += uint64(sz)
		//T().Debugf("sc.buffer = '%s'", string(sc.buffer))
	}
	if len(sc.lookahead) > 0 { // process left-over LA
		// sc.prepareNewRun()
		sc.lookahead = sc.lookahead[:0]
		//sc.pos = sc.ahead // catch up new input position
		//T().Debugf("+ LEN=%d, POS=%d", sc.ahead, sc.pos)
		//r, clz, sz := sc.bidic(sc.buffer) // calculate current bidi class
		r, clz, _ := sc.bidic(sc.buffer) // re-calculate current bidi class; size already done
		sc.bidiclz, openpos, isBracket = sc.doBD16(r, sc.ahead, clz, true)
		//sc.bidiclz = clz
		//sc.ahead += uint64(sz) // include len(LA) in run's ahead
		T().Debugf("final Token '%s' as :%s", string(sc.buffer), ClassString(sc.bidiclz))
		//T().Debugf("final Token '%s' as :%s", string(sc.buffer), ClassString(sc.bidiclz))
		l := uint64(len(sc.buffer)) // length of current bidi cluster
		if isBracket {
			//_, openpos, _ = sc.doBD16(r, sc.ahead, clz, true)
			//if openpos != sc.ahead {
			if sc.bidiclz == BRACKC {
				T().Debugf("position of opening bracket is %d", openpos)
				// We have the closing bracket to a corresponding open one.
				// Brackets are always of length 1, so we'll misuse the length field
				// for some other information: the position of the openending bracket
				// if this is a closing bracket.
				l = uint64(openpos)
			}
			panic("bracket")
		}
		if sc.bidiclz == bidi.AL {
			sc.bidiclz = bidi.R // rule W3
		}
		return int(sc.bidiclz), sc.strtyps, sc.pos, l
	}
	if !sc.done {
		sc.done = true
		T().Debugf("final synthetic Token :%s", ClassString(bidi.PDI))
		return int(bidi.PDI), sc.strtyps, sc.pos, 0
	}
	return scanner.EOF, sc.strtyps, sc.pos, 0
}

func (sc *bidiScanner) prepareNewRun() {
	sc.pos = sc.ahead // catch up new input position
	if len(sc.lookahead) > 0 {
		sc.buffer = sc.buffer[:0]                             // reset buffer
		sc.buffer = append(sc.buffer, sc.lookahead...)        // move LA to buffer
		r, clz, sz := sc.bidic(sc.buffer)                     // calculate current bidi class
		sc.bidiclz, _, _ = sc.doBD16(r, sc.ahead, clz, false) // check for brackets
		sc.ahead += uint64(sz)                                // include len(LA) in run's ahead
	}
	//T().Debugf("- LEN=%d, POS=%d", sc.ahead, sc.pos)
}
*/

// SetErrorHandler sets an error handler function, which receives an error
// and may try some error repair strategy.
//
// Currently does nothing.
//
func (sc *bidiScanner) SetErrorHandler(h func(error)) {
	// TODO
}

// --- Handling of BiDi Classes ----------------------------------------------

// bidic returns the Bidi_Class for a rune. It will apply certain UAX#9 rules
// immediately to relief the parser.
//
// TODO Completely implement W1 on scanner level
//
func (sc *bidiScanner) bidic(r rune) bidi.Class {
	//r, sz := utf8.DecodeRune(b)
	//if sz > 0 {
	if sc.hasMode(optionTesting) && unicode.IsUpper(r) {
		return bidi.R // during testing, UPPERCASE is R2L
	}
	// props, sz := bidi.Lookup(b)
	// clz := props.Class()
	switch clz { // do some pre-processing
	case bidi.NSM: // rule W1, handle accents
		switch sc.bidiclz {
		case bidi.LRI:
			return bidi.L
		case bidi.RLI:
			return bidi.R
		case bidi.PDI:
			//return r, bidi.ON, sz
			return NI
		}
		return sc.bidiclz
	case bidi.EN: // rule W2
		if sc.strtyps.IsAL() {
			//case bidi.AL:
			return bidi.AN
			// case bidi.L:
			// 	return r, LEN, sz
		}
	//case bidi.AL: // rule W3 //return r, bidi.R, sz
	case bidi.S, bidi.WS, bidi.ON:
		return NI
		//if sc.bidiclz == NI {
		// return NI
		//}
	}
	return props.Class()
}

func (sc *bidiScanner) doBD16(r rune, pos charpos, defaultclass bidi.Class, doStack bool) (
	bidi.Class, charpos, bool) {
	//
	if !doStack {
		//T().Debugf("checking bidi class of %v", r)
		c, isbr := sc.checkBD16(r, defaultclass)
		return c, -1, isbr
	}
	var isbr bool
	var openpos int64
	if isbr, sc.bd16stack = sc.bd16stack.pushIfBracket(r, pos); isbr {
		T().Debugf("BD16 - pushed an opening bracket: %v", r)
		switch sc.strtyps.Context() {
		case bidi.L:
			return LBRACKO, int64(pos), true
		case bidi.R:
			return RBRACKO, int64(pos), true
		}
		return LBRACKO, int64(pos), true
	} else if isbr, openpos, sc.bd16stack = sc.bd16stack.popWith(r, pos); isbr {
		T().Debugf("BD16 - popped a closing bracket: %v", r)
		//panic("DB16")
		return BRACKC, openpos, true
	}
	return defaultclass, -1, false
}

func (sc *bidiScanner) checkBD16(r rune, defaultclass bidi.Class) (bidi.Class, bool) {
	props, _ := bidi.LookupRune(r)
	if props.IsBracket() {
		//T().Debugf("Bracket detected: %c", r)
		//T().Debugf("Bracket '%c' with sc.strong = %s", r, ClassString(sc.strong))
		switch sc.strtyps.Context() {
		case bidi.L:
			if props.IsOpeningBracket() {
				return LBRACKO, true
			}
			return BRACKC, true
		case bidi.R:
			if props.IsOpeningBracket() {
				return RBRACKO, true
			}
			return BRACKC, true
		}
		return LBRACKO, true
	}
	return defaultclass, false
}

func (sc *bidiScanner) setStrongPos(c bidi.Class) {
	switch c {
	case bidi.L, bidi.LRI:
		sc.strtyps.SetLDist(int(sc.pos))
	case bidi.R, bidi.RLI:
		sc.strtyps.SetRDist(int(sc.pos))
	case bidi.AL:
		sc.strtyps.SetALDist(int(sc.pos))
	}
}

func (sc *bidiScanner) Context() bidi.Class {
	return sc.strtyps.Context()
}

// --- Bidi_Classes ----------------------------------------------------------

// We use some additional Bidi_Classes, which reflect additional knowledge about
// a character(-sequence). Our scanner will process some BiDi rules before the parser is
// going to see the tokens.
//
// Unfortunately we need the additional BiDi-classes to be close to the ones defined in package unicode.bidi,
// to fit them in a compact hash trie. This creates an unwanted dependency on the maximum value of
// BiDi classes in unicode.bidi, which as of now is `bidi.PDI`. Package unicode.bidi is
// unstable, thus making us somewhat reliant on an unreliable API.
const (
	LBRACKO bidi.Class = bidi.PDI + 1 + iota // opening bracket in L context
	RBRACKO                                  // opening bracket in R context
	BRACKC                                   // closing bracket
	NI                                       // neutral character
	MAX                                      // marker to have the maximum BiDi class available for clients
	ILLEGAL bidi.Class = 999                 // in-band value denoting illegal class
	//LBRACKC                                  // closing bracket in L context
	//RBRACKC                                  // closing bracket in R context
)

const claszname = "LRENESETANCSBSWSONBNNSMALControlNumLRORLOLRERLEPDFLRIRLIFSIPDI----------"
const claszadd = "LBRACKORBRACKOBRACKCNI<max>------"

var claszindex = [...]uint8{0, 1, 2, 4, 6, 8, 10, 12, 13, 14, 16, 18, 20, 23, 25, 32, 35, 38, 41, 44, 47, 50, 53, 56, 59, 62}
var claszaddinx = [...]uint8{0, 7, 14, 20, 22, 27}

// ClassString returns a bidi class as a string.
func ClassString(i bidi.Class) string {
	if i == ILLEGAL {
		return "bidi_class(none)"
	}
	if i > bidi.PDI {
		if i > bidi.PDI && i <= MAX {
			j := i - LBRACKO
			return claszadd[claszaddinx[j]:claszaddinx[j+1]]
		}
		return "bidi_class(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return claszname[claszindex[i]:claszindex[i+1]]
}

// --- Brackets and bracket stack --------------------------------------------

type bracketStack []bpos
type bpos struct {
	pos  uint64
	pair bracketPair
}

func (bs bracketStack) push(b rune, pos uint64) (bool, bracketStack) {
	if len(bs) == BS16Max { // skip in case of stack overflow, as def in UAX#9
		return false, bs
	}
	for _, pair := range uax9BracketPairs { // double check for UAX#9 brackets
		if pair.o == b {
			b := bpos{pos: pos, pair: pair}
			return true, append(bs, b)
		}
	}
	T().Errorf("Push of %c failed, not found as opening bracket")
	return false, bs
}

func (bs bracketStack) pushIfBracket(b rune, pos uint64) (bool, bracketStack) {
	props, _ := bidi.LookupRune(b)
	if props.IsBracket() && props.IsOpeningBracket() {
		T().Errorf("pushing bracket %v, bracket stack was %v", b, bs)
		return bs.push(b, pos)
	}
	return false, bs
}

func (bs bracketStack) popWith(b rune, pos uint64) (bool, int64, bracketStack) {
	T().Errorf("popWith: rune=%v, bracket stack is %v", b, bs)
	if len(bs) == 0 {
		return false, -1, bs
	}
	i := len(bs) - 1
	for i >= 0 { // start at TOS, possible skip unclosed opening brackets
		if bs[i].pair.c == b {
			openpos := bs[i].pos
			bs = bs[:i]
			return true, int64(openpos), bs
		}
		i--
	}
	return false, -1, bs
}

func (bs bracketStack) dump() {
	for i, p := range bs {
		T().Debugf("\t[%d] %v at %d", i, p.pair, p.pos)
	}
}

// --- Scanner options -------------------------------------------------------

// Option configures a bidi scanner
type Option func(p *bidiScanner)

const (
	optionRecognizeLegacy uint = 1 << 1 // recognize LRM, RLM, ALM, LRE, RLE, LRO, RLO, PDF
	optionOuterR2L        uint = 1 << 2 // set outer direction as RtoL
	optionTesting         uint = 1 << 3 // test mode: recognize uppercase as class R
)

// RecognizeLegacy is not yet implemented. It was indented to make the
// resolver recognize legacy formatting, i.e.
// LRM, RLM, ALM, LRE, RLE, LRO, RLO, PDF. However, I changed my mind and
// currently do not intend to support legacy formatting types,
// thus setting this option will have no effect.
func RecognizeLegacy(b bool) Option {
	return func(sc *bidiScanner) {
		if !sc.hasMode(optionRecognizeLegacy) && b ||
			sc.hasMode(optionRecognizeLegacy) && !b {
			sc.mode |= optionRecognizeLegacy
		}
	}
}

// Testing will set up the scanner to recognize UPPERCASE letters as having R2L class.
// This is a common pattern in bidi algorithm development.
func Testing(b bool) Option {
	return func(sc *bidiScanner) {
		if !sc.hasMode(optionTesting) && b ||
			sc.hasMode(optionTesting) && !b {
			sc.mode |= optionTesting
		}
	}
}

func (sc *bidiScanner) hasMode(m uint) bool {
	return sc.mode&m > 0
}
