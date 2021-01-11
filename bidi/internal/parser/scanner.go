package parser

import (
	"bufio"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/npillmayer/gorgo/lr/scanner"
	"golang.org/x/text/unicode/bidi"
)

// Scanner implements the scanner.Tokenizer interface.
// It will read runs of text as a unit, as long as all runes therein have the
// same Bidi_Class.
type Scanner struct {
	runeScanner *bufio.Scanner // we're using an embedded rune reader
	currClz     bidi.Class     // the current Bidi_Class (of the last rune read)
	lookahead   []byte         // lookahead rune
	buffer      []byte         // character buffer for token lexeme
	strong      bidi.Class     // Bidi_Class of last strong character encountered
	pos         uint64         // position in input string
	ahead       uint64         // position ahead of current lexeme
	bd16stack   bracketStack   // bracket pair stack, rule BD16
	done        bool           // at EOF?
	mode        uint           // scanner modes, set with options
}

// BS16Max is the maximum stack depth for rule BS16 as defined in UAX#9.
const BS16Max = 63
const buflen = 4096

// NewScanner creates a scanner for bidi formatting. It will read runs of text
// as a unit, as long as all runes therein have the same Bidi_Class.
//
// Clients will provide a Reader and zero or more scanner options. Runes will be
// read from the reader and possibly concatenated to chunks of text with
// identical BiDi class (see NextToken).
func NewScanner(input io.Reader, opts ...ScannerOption) *Scanner {
	sc := &Scanner{}
	sc.runeScanner = bufio.NewScanner(input)
	sc.runeScanner.Split(bufio.ScanRunes)
	sc.currClz = bidi.LRI
	sc.buffer = make([]byte, 0, buflen)
	sc.lookahead = make([]byte, 0, 32)
	sc.bd16stack = make(bracketStack, 0, BS16Max+1)
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

// NextToken reads the next run of input text with identical bidi_class,
// returning a token for it.
//
// The token's value will be set to the bidi_class, the token itself will be
// set to the corresponding input string.
func (sc *Scanner) NextToken(expected []int) (int, interface{}, uint64, uint64) {
	sc.prepareNewRun()
	//T().Debugf("re-reading '%s'", string(sc.buffer))
	for sc.runeScanner.Scan() {
		b := sc.runeScanner.Bytes()
		//T().Debugf("--------------")
		r, clz, sz := sc.bidic(b)
		//T().Debugf("next char '%s' has class %s", string(b), ClassString(clz))
		var isBracket bool
		clz, isBracket = sc.doBD16(r, sc.ahead, clz, true)
		if clz != sc.currClz || isBracket { // bidi classes get collected, but brackets dont't
			sc.lookahead = sc.lookahead[:0]
			sc.lookahead = append(sc.lookahead, b...)
			rc := sc.currClz // tmp for returning current class
			sc.currClz = clz // change current class to class of LA
			T().Debugf("scanned Token '%s' as :%s", string(sc.buffer), ClassString(rc))
			return int(rc), sc.buffer, sc.pos, uint64(len(sc.buffer))
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
		sc.currClz, _ = sc.doBD16(r, sc.ahead, clz, false)
		//sc.currClz = clz
		//sc.ahead += uint64(sz) // include len(LA) in run's ahead
		T().Debugf("final Token '%s' as :%s", string(sc.buffer), ClassString(sc.currClz))
		return int(sc.currClz), sc.buffer, sc.pos, uint64(len(sc.buffer))
	}
	if !sc.done {
		sc.done = true
		T().Debugf("final synthetic Token :%s", ClassString(bidi.PDI))
		return int(bidi.PDI), "", sc.pos, 0
	}
	return scanner.EOF, "", sc.pos, 0
}

func (sc *Scanner) prepareNewRun() {
	sc.pos = sc.ahead // catch up new input position
	if len(sc.lookahead) > 0 {
		sc.buffer = sc.buffer[:0]                          // reset buffer
		sc.buffer = append(sc.buffer, sc.lookahead...)     // move LA to buffer
		r, clz, sz := sc.bidic(sc.buffer)                  // calculate current bidi class
		sc.currClz, _ = sc.doBD16(r, sc.ahead, clz, false) // check for brackets
		sc.ahead += uint64(sz)                             // include len(LA) in run's ahead
	}
	//T().Debugf("- LEN=%d, POS=%d", sc.ahead, sc.pos)
}

// SetErrorHandler sets an error handler function, which receives an error
// and may try some error repair strategy.
//
// Currently does nothing.
//
func (sc *Scanner) SetErrorHandler(h func(error)) {
	// TODO
}

// --- Handling of BiDi Classes ----------------------------------------------

// bidic returns the Bidi_Class for a rune. It will apply certain UAX#9 rules
// immediately to relief the parser.
//
// TODO Completely implement W1 on scanner level
//
func (sc *Scanner) bidic(b []byte) (rune, bidi.Class, int) {
	r, sz := utf8.DecodeRune(b)
	if sz > 0 {
		if sc.hasMode(optionTesting) && unicode.IsUpper(r) {
			sc.setIfStrong(bidi.R)
			return r, bidi.R, sz // during testing, UPPERCASE is R2L
		}
		props, sz := bidi.Lookup(b)
		clz := props.Class()
		sc.setIfStrong(clz)
		switch clz { // do some pre-processing
		case bidi.NSM: // rule W1, handle accents
			switch sc.currClz {
			case bidi.LRI:
				return r, bidi.L, sz
			case bidi.RLI:
				return r, bidi.R, sz
			case bidi.PDI:
				return r, bidi.ON, sz
			}
			return r, sc.currClz, sz
		case bidi.EN: // rule W2 and pretext to W7
			if sc.currClz == bidi.L {
				return r, bidi.L, sz
			}
			switch sc.strong {
			case bidi.AL:
				return r, bidi.AN, sz
			case bidi.L:
				return r, LEN, sz
			}
		case bidi.S:
			fallthrough
		case bidi.WS:
			return r, NI, sz
		case bidi.ON:
			if sc.currClz == NI {
				return r, NI, sz
			}
		}
		return r, props.Class(), sz
	}
	return 0, bidi.L, 0
}

func (sc *Scanner) doBD16(r rune, pos uint64, defaultclass bidi.Class, doStack bool) (bidi.Class, bool) {
	if !doStack {
		return sc.checkBD16(r, defaultclass)
	}
	var isbr bool
	if isbr, sc.bd16stack = sc.bd16stack.pushIfBracket(r, pos); isbr {
		T().Debugf("BD16 - pushed an opening bracket: %v", r)
		switch sc.strong {
		case bidi.L:
			return LBRACKO, true
		case bidi.R:
			return RBRACKO, true
		}
		return LBRACKO, true
	} else if isbr, sc.bd16stack = sc.bd16stack.popWith(r, pos); isbr {
		T().Debugf("BD16 - popped a closing bracket: %v", r)
		return BRACKC, true
	}
	return defaultclass, false
}

func (sc *Scanner) checkBD16(r rune, defaultclass bidi.Class) (bidi.Class, bool) {
	props, _ := bidi.LookupRune(r)
	if props.IsBracket() {
		//T().Debugf("Bracket detected: %c", r)
		//T().Debugf("Bracket '%c' with sc.strong = %s", r, ClassString(sc.strong))
		switch sc.strong {
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

func (sc *Scanner) setIfStrong(c bidi.Class) bidi.Class {
	switch c {
	case bidi.R, bidi.RLI:
		sc.strong = bidi.R
		return bidi.R
	case bidi.L, bidi.LRI:
		sc.strong = bidi.L
		return bidi.L
	case bidi.AL:
		sc.strong = bidi.AL
		return bidi.AL
	}
	return ILLEGAL
}

// --- Bidi_Class Helpers ----------------------------------------------------

// We use some additional Bidi_Classes, which reflects additional knowledge about
// a character. Our scanner will process some Bidi rules before the parser is
// going to see the tokens.
const (
	LEN     bidi.Class = bidi.PDI + 1 // left biased european number (EN)
	LBRACKO                           // opening bracket in L context
	RBRACKO                           // opening bracket in R context
	LBRACKC                           // closing bracket in L context
	RBRACKC                           // closing bracket in R context
	BRACKC                            // closing bracket
	NI                                // neutral character
	ILLEGAL bidi.Class = 999          // in-band value denoting illegal class
)

const claszname = "LRENESETANCSBSWSONBNNSMALControlNumLRORLOLRERLEPDFLRIRLIFSIPDI----------"
const claszadd = "LENLBRACKORBRACKOLBRACKCRBRACKCBRACKCNI-----------"

var claszindex = [...]uint8{0, 1, 2, 4, 6, 8, 10, 12, 13, 14, 16, 18, 20, 23, 25, 32, 35, 38, 41, 44, 47, 50, 53, 56, 59, 62}
var claszaddinx = [...]uint8{0, 3, 10, 17, 24, 31, 37, 39}

// ClassString returns a bidi class as a string.
func ClassString(i bidi.Class) string {
	if i == ILLEGAL {
		return "bidi_class(none)"
	}
	if i >= bidi.Class(len(claszindex)-1) {
		if i >= LEN && i < LEN+bidi.Class(len(claszaddinx)) {
			j := i - LEN
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
	pair BracketPair
}

func (bs bracketStack) push(b rune, pos uint64) (bool, bracketStack) {
	if len(bs) == BS16Max { // skip in case of stack overflow, as def in UAX#9
		return false, bs
	}
	for _, pair := range UAX9BracketPairs { // double check for UAX#9 brackets
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
		return bs.push(b, pos)
	}
	return false, bs
}

func (bs bracketStack) popWith(b rune, pos uint64) (bool, bracketStack) {
	if len(bs) == 0 {
		return false, bs
	}
	i := len(bs) - 1
	for i >= 0 { // start at TOS, possible skip unclosed opening brackets
		if bs[i].pair.c == b {
			bs = bs[:i]
			return true, bs
		}
		i--
	}
	return false, bs
}

func (bs bracketStack) dump() {
	for i, p := range bs {
		T().Debugf("\t[%d] %v at %d", i, p.pair, p.pos)
	}
}

// --- Scanner options -------------------------------------------------------

// ScannerOption configures a bidi scanner
type ScannerOption func(p *Scanner)

const (
	optionRecognizeLegacy uint = 1 << 1 // recognize LRM, RLM, ALM, LRE, RLE, LRO, RLO, PDF
	optionOuterR2L        uint = 1 << 2 // set outer direction as RtoL
	optionTesting         uint = 1 << 3 // test mode: recognize uppercase as class R
)

// RecognizeLegacy sets an option to recognize legacy formatting, i.e.
// LRM, RLM, ALM, LRE, RLE, LRO, RLO, PDF.
func RecognizeLegacy(b bool) ScannerOption {
	return func(sc *Scanner) {
		if !sc.hasMode(optionRecognizeLegacy) && b ||
			sc.hasMode(optionRecognizeLegacy) && !b {
			sc.mode |= optionRecognizeLegacy
		}
	}
}

// Testing will set up the scanner to recognize UPPERCASE letters as having R2L class.
// This is a common pattern in bidi algorithm development.
func Testing(b bool) ScannerOption {
	return func(sc *Scanner) {
		if !sc.hasMode(optionTesting) && b ||
			sc.hasMode(optionTesting) && !b {
			sc.mode |= optionTesting
		}
	}
}

func (sc *Scanner) hasMode(m uint) bool {
	return sc.mode&m > 0
}
