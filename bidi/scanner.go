package bidi

// Notes:
// - Input reader will in most cases be a cord reader

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
	mode uint8 // scanner modes, set by scanner options
	// bidiclz     bidi.Class          // the current bidi class
	// lookahead   scrap               // lookahead has been read but not sent to parser
	// pos         charpos             // position in input string
	// ahead       charpos             // position ahead of current scrap
	// strongs     strongTypes         // positions of previously occured strong types
	runeScanner *bufio.Scanner      // we're using an embedded rune reader
	bd16        *bracketPairHandler // support type for handling bracket pairings
	// bd16stack   bracketStack        // bracket pair stack, rule BD16
	//laLen       charpos        // byte length of lookahead
	// lookahead   []byte         // lookahead rune
	// buffer      []byte         // character buffer for token lexeme
}

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
	//sc.bidiclz = bidi.LRI
	//sc.buffer = make([]byte, 0, buflen)
	//sc.lookahead = make([]byte, 0, 32s)
	sc.bd16 = makeBracketPairHandler(nil) // TODO handle nested isolate runs
	//sc.bd16stack = make(bracketStack, 0, BS16Max+1)
	for _, opt := range opts {
		if opt != nil {
			opt(sc)
		}
	}
	return sc
}

// nextRune reads the next rune from the input reader.
// Returns the rune, its byte length, bidi class and a flag indicating
// a valid input (false for EOF).
func (sc *bidiScanner) nextRune() (rune, int, bidi.Class, bool) {
	if ok := sc.runeScanner.Scan(); !ok {
		return 0, 0, cNULL, false
	}
	b := sc.runeScanner.Bytes()
	r, length := utf8.DecodeRune(b)
	props, sz := bidi.LookupRune(r)
	if sz == 0 || sz != length {
		panic("bidi package differs in rune interpretation from scanner package")
	}
	bidiclz := props.Class()
	if props.IsBracket() {
		if props.IsOpeningBracket() {
			bidiclz = cBRACKO
		} else {
			bidiclz = cBRACKC
		}
	}
	T().Debugf("scanner rune %#U (%s)", r, classString(bidiclz))
	return r, length, bidiclz, true
}

// makeScrap wraps the input rune into a scrap.
func makeScrap(r rune, clz bidi.Class, pos charpos, length int) scrap {
	s := scrap{
		l:       pos,
		r:       pos + charpos(length),
		bidiclz: clz,
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
	//var pos, lapos charpos // position of current scrap and of lookahead
	var lookahead scrap
	current := sc.initCurrentScrap()
	var lastAL charpos // position of last AL character-run in input
	for {
		r, length, bidiclz, ok := sc.nextRune() // read the next input rune
		if !ok {                                // if EOF, drain lookahead and quit
			sc.post(lookahead, pipe) // use lookahead as current scrap and put it onto pipe
			sc.stop(pipe)            // then send quit signal to parser
			break
		}

		// a rune has successfully been read ⇒ make it the new lookahead
		if sc.hasMode(optionTesting) && unicode.IsUpper(r) {
			bidiclz = bidi.R // during testing, UPPERCASE is R2L
		}
		isAL := bidiclz == bidi.AL                     // AL will be changed by rule W3
		bidiclz = applyRulesW1to3(r, bidiclz, current) // UAX#9 W1–3 handled by scanner
		//lookahead = makeScrap(r, bidiclz, lapos, length)
		lookahead = makeScrap(r, bidiclz, current.r, length)
		T().Debugf("bidi scanner lookahead = %v", lookahead) // finally a new lookahead

		// the current scrap is finished if the lookahead cannot extend it
		if current.bidiclz == cNULL || current.bidiclz != lookahead.bidiclz || isbracket(current) {
			if current.bidiclz != cNULL { // if the current scrap is not the initial null scrap
				sc.post(current, pipe) // put current on channel
			}
			// proceed ahead, making lookahead the current scrap
			inheritStrongTypes(lookahead, current, lastAL)
			current = sc.prepareRuleBD16(r, lookahead)
			//current = lookahead
			// pos, lapos = lapos, lookahead.r
			// if pos > 0 {
			// 	// TOTO remove pos?
			// }
			if isAL {
				lastAL = current.l
			}
		} else { // otherwise the current scrap grows
			//lapos = lookahead.r
			current = collapse(current, lookahead, current.bidiclz) // merge LA with current
			T().Debugf("current = %s, next iteration", current)
		}
	}
	T().Infof("stopped bidi scanner")
}

func (sc *bidiScanner) post(s scrap, pipe chan<- scrap) {
	T().Debugf("bidi scanner sends current scrap: %v", s)
	pipe <- s
}

func (sc *bidiScanner) stop(pipe chan<- scrap) {
	T().Debugf("stopping bidi scanner, sending final scrap (stopper)")
	s := scrap{bidiclz: cNULL}
	pipe <- s
	close(pipe)
}

func (sc *bidiScanner) initCurrentScrap() scrap {
	var current scrap
	current.bidiclz = cNULL
	if sc.hasMode(optionOuterR2L) {
		current.context.SetEmbedding(bidi.RightToLeft)
	} else {
		current.context.SetEmbedding(bidi.LeftToRight)
	}
	return current
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
		//T().Debugf("next char '%s' has class %s", string(b), classString(clz))
		//clz, _, isBracket = sc.doBD16(r, sc.ahead, clz, false)
		clz, isBracket := sc.checkBD16(r, clz)
		if clz != sc.bidiclz || isBracket { // bidi classes get collected, but brackets dont't
			sc.lookahead = sc.lookahead[:0]           // truncate previous LA
			sc.lookahead = append(sc.lookahead, b...) // and replace by last read rune
			current := sc.bidiclz                     // tmp for returning current class
			sc.bidiclz = clz                          // change current class to class of LA
			sc.setStrongPos(current)                  // remember if current is a strong type
			T().Debugf("scanned Token '%s' as :%s", string(sc.buffer), classString(current))
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
		T().Debugf("final Token '%s' as :%s", string(sc.buffer), classString(sc.bidiclz))
		//T().Debugf("final Token '%s' as :%s", string(sc.buffer), classString(sc.bidiclz))
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
		T().Debugf("final synthetic Token :%s", classString(bidi.PDI))
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

// applyRulesW1to3 returns the Bidi_Class for a rune. It will apply certain UAX#9
// rules immediately to relief the parser.
//
func applyRulesW1to3(r rune, clz bidi.Class, current scrap) bidi.Class {
	//r, sz := utf8.DecodeRune(b)
	//if sz > 0 {
	// props, sz := bidi.Lookup(b)
	// clz := props.Class()
	currclz := current.bidiclz
	switch clz { // do some pre-processing
	case bidi.NSM: // rule W1, handle accents
		switch currclz {
		case bidi.LRI:
			return bidi.L
		case bidi.RLI:
			return bidi.R
		case bidi.PDI:
			//return r, bidi.ON, sz
			return cNI
		}
		return currclz
	case bidi.EN: // rule W2
		//if sc.strtyps.IsAL() {
		if current.context.IsAL() {
			//case bidi.AL:
			return bidi.AN
			// case bidi.L:
			// 	return r, LEN, sz
		}
	case bidi.AL: // rule W3
		return bidi.R
	case bidi.S, bidi.WS, bidi.ON:
		return cNI
		//if sc.bidiclz == NI {
		// return NI
		//}
	}
	//return props.Class()
	return clz
}

func (sc *bidiScanner) prepareRuleBD16(r rune, s scrap) scrap {
	if !isbracket(s) {
		return s
	}
	if s.bidiclz == cBRACKO {
		//var isbr bool
		// is LA not just a bracket, but part of a UAX#9 bracket pair?
		isbr := sc.bd16.pushOpening(r, s)
		//isbr, sc.bd16stack = sc.bd16stack.push(r, la.l)
		if isbr {
			T().Debugf("pushed lookahead onto bracket stack: %s", s)
			sc.bd16.dump()
		}
	} else {
		found, _ := sc.bd16.findPair(r, s)
		if found {
			T().Debugf("popped closing bracket: %s", s)
			sc.bd16.dump()
		}
	}
	// clz, openbrpos, isbr := sc.doBD16(r, pos, s.bidiclz, true)
	// if isbr {
	// 	s.bidiclz = clz
	// 	if openbrpos >= 0 {
	// 		s.r = openbrpos // bracket scraps mis-use r for position of opening bracket
	// 	}
	// }
	return s
}

// func (sc *bidiScanner) doBD16(r rune, pos charpos, defaultclass bidi.Class, doStack bool) (
// 	bidi.Class, charpos, bool) {
// 	//
// 	if !doStack {
// 		//T().Debugf("checking bidi class of %v", r)
// 		c, isbr := sc.checkBD16(r, defaultclass)
// 		return c, -1, isbr
// 	}
// 	var isbr bool
// 	var openpos int64
// 	if isbr, sc.bd16stack = sc.bd16stack.pushIfBracket(r, pos); isbr {
// 		T().Debugf("BD16 - pushed an opening bracket: %v", r)
// 		switch sc.strtyps.Context() {
// 		case bidi.L:
// 			return LBRACKO, int64(pos), true
// 		case bidi.R:
// 			return RBRACKO, int64(pos), true
// 		}
// 		return LBRACKO, int64(pos), true
// 	} else if isbr, openpos, sc.bd16stack = sc.bd16stack.popWith(r, pos); isbr {
// 		T().Debugf("BD16 - popped a closing bracket: %v", r)
// 		//panic("DB16")
// 		return BRACKC, openpos, true
// 	}
// 	return defaultclass, -1, false
// }

// func (sc *bidiScanner) checkBD16(r rune, defaultclass bidi.Class) (bidi.Class, bool) {
// 	props, _ := bidi.LookupRune(r)
// 	if props.IsBracket() {
// 		//T().Debugf("Bracket detected: %c", r)
// 		//T().Debugf("Bracket '%c' with sc.strong = %s", r, classString(sc.strong))
// 		switch sc.strtyps.Context() {
// 		case bidi.L:
// 			if props.IsOpeningBracket() {
// 				return LBRACKO, true
// 			}
// 			return BRACKC, true
// 		case bidi.R:
// 			if props.IsOpeningBracket() {
// 				return RBRACKO, true
// 			}
// 			return BRACKC, true
// 		}
// 		return LBRACKO, true
// 	}
// 	return defaultclass, false
// }

// isAL is true if dest has been of bidi class AL (before UAX#9 rule W3 changed it)
func inheritStrongTypes(dest, src scrap, lastAL charpos) {
	dest.context = src.context
	dest.context.SetStrongType(bidi.AL, lastAL)
	switch src.bidiclz {
	case bidi.L, bidi.LRI:
		dest.context.SetStrongType(bidi.L, src.l)
	case bidi.R, bidi.RLI:
		dest.context.SetStrongType(bidi.R, src.l)
	case bidi.AL:
		dest.context.SetStrongType(bidi.AL, src.l)
	}
}

// func (sc *bidiScanner) setStrongPos(c bidi.Class) {
// 	switch c {
// 	case bidi.L, bidi.LRI:
// 		sc.strtyps.SetLDist(int(sc.pos))
// 	case bidi.R, bidi.RLI:
// 		sc.strtyps.SetRDist(int(sc.pos))
// 	case bidi.AL:
// 		sc.strtyps.SetALDist(int(sc.pos))
// 	}
// }

// func (sc *bidiScanner) Context() bidi.Class {
// 	return sc.strtyps.Context()
// }

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
	cBRACKO bidi.Class = bidi.PDI + 1 + iota // opening bracket
	cBRACKC                                  // closing bracket
	cNI                                      // neutral character
	cMAX                                     // marker to have the maximum BiDi class available for clients
	cNULL   bidi.Class = 999                 // in-band value denoting illegal class
	//RBRACKO                                  // opening bracket in R context
	//LBRACKC                                  // closing bracket in L context
	//RBRACKC                                  // closing bracket in R context
)

const claszname = "LRENESETANCSBSWSONBNNSMALControlNumLRORLOLRERLEPDFLRIRLIFSIPDI----------"
const claszadd = "BRACKOBRACKCNI<max>------"

var claszindex = [...]uint8{0, 1, 2, 4, 6, 8, 10, 12, 13, 14, 16, 18, 20, 23, 25, 32, 35, 38, 41, 44, 47, 50, 53, 56, 59, 62}
var claszaddinx = [...]uint8{0, 6, 12, 14, 19, 20}

// classString returns a bidi class as a string.
func classString(i bidi.Class) string {
	if i == cNULL {
		return "cNULL"
	}
	if i > bidi.PDI {
		if i > bidi.PDI && i <= cMAX {
			j := i - cBRACKO
			return claszadd[claszaddinx[j]:claszaddinx[j+1]]
		}
		return "bidi_class(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return claszname[claszindex[i]:claszindex[i+1]]
}

// --- Scanner options -------------------------------------------------------

// Option configures a bidi scanner
type Option func(p *bidiScanner)

const (
	optionRecognizeLegacy uint8 = 1 << 1 // recognize LRM, RLM, ALM, LRE, RLE, LRO, RLO, PDF
	optionOuterR2L        uint8 = 1 << 2 // set outer direction as RtoL
	optionTesting         uint8 = 1 << 3 // test mode: recognize uppercase as class R
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

// DefaultDirection sets outer embedding level for a paragraph
// (LeftToRight is the normal default).
func DefaultDirection(dir Direction) Option {
	if dir == RightToLeft {
		return func(sc *bidiScanner) {
			sc.mode |= optionOuterR2L
		}
	}
	return nil
}

// TestMode will set up the scanner to recognize UPPERCASE letters as having R2L class.
// This is a common pattern in bidi algorithm development and testing.
func TestMode(b bool) Option {
	return func(sc *bidiScanner) {
		if !sc.hasMode(optionTesting) && b ||
			sc.hasMode(optionTesting) && !b {
			sc.mode |= optionTesting
		}
	}
}

func (sc *bidiScanner) hasMode(m uint8) bool {
	return sc.mode&m > 0
}
