package bidi

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
// It will read runs of text as a unit, as long as all runes therein have the same Bidi_Class.
type Scanner struct {
	runeScanner *bufio.Scanner // we're using an embedded rune reader
	currClz     bidi.Class     // the current Bidi_Class (of the last rune read)
	lookahead   []byte         // lookahead rune
	buffer      []byte         // character buffer for token lexeme
	brackets    bracketStack   // stack for correlating bracket characters
	strong      bidi.Class     // Bidi_Class of last strong character encountered
	pos         uint64         // position in input string
	length      uint64         // length of current lexeme
	done        bool           // at EOF?
	mode        uint           // scanner modes, set with options
}

// NewScanner creates a scanner for bidi formatting. It will read runs of text
// as a unit, as long as all runes therein have the same Bidi_Class.
func NewScanner(input io.Reader, opts ...ScannerOption) *Scanner {
	sc := &Scanner{}
	sc.runeScanner = bufio.NewScanner(input)
	sc.runeScanner.Split(bufio.ScanRunes)
	sc.currClz = bidi.LRI
	sc.buffer = make([]byte, 0, 4096)
	sc.lookahead = make([]byte, 0, 32)
	sc.brackets = make([]BracketPair, 0, 16)
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
	if len(sc.lookahead) > 0 {
		sc.prepareNewRun()
		//T().Debugf("re-reading '%s'", string(sc.buffer))
	}
	for sc.runeScanner.Scan() {
		b := sc.runeScanner.Bytes()
		//T().Debugf("--------------")
		clz, sz := sc.bidic(b)
		//T().Debugf("next char '%s' has class %s", string(b), ClassString(clz))
		if clz != sc.currClz {
			sc.lookahead = sc.lookahead[:0]
			sc.lookahead = append(sc.lookahead, b...)
			r := sc.currClz  // tmp for returning current class
			sc.currClz = clz // change current class to class of LA
			// TODO check with bracket stack !
			//r = sc.replaceIfMatchingBrackets(b)
			T().Debugf("scanned Token '%s' as :%s", string(sc.buffer), ClassString(r))
			return int(r), sc.buffer, sc.pos, uint64(len(sc.buffer))
		}
		sc.buffer = append(sc.buffer, b...)
		sc.length += uint64(sz)
		//T().Debugf("sc.buffer = '%s'", string(sc.buffer))
	}
	if len(sc.lookahead) > 0 {
		// sc.prepareNewRun()
		sc.lookahead = sc.lookahead[:0]
		sc.pos += sc.length            // set new input position
		clz, sz := sc.bidic(sc.buffer) // calculate current bidi class
		sc.currClz = clz
		sc.length += uint64(sz) // include len(LA) in run's length
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

func (sc *Scanner) replaceIfMatchingBrackets(b rune) bidi.Class {
	// if isbr, sc.brackets = sc.brackets.pushIfBracket(r); isbr {
	return bidi.ON // TODO
}

func (sc *Scanner) prepareNewRun() {
	var sz int
	sc.buffer = sc.buffer[:0]                      // reset buffer
	sc.buffer = append(sc.buffer, sc.lookahead...) // move LA to buffer
	sc.pos += sc.length                            // set new input position
	sc.currClz, sz = sc.bidic(sc.buffer)           // calculate current bidi class
	sc.length += uint64(sz)                        // include len(LA) in run's length
}

// bidic returns the Bidi_Class for a rune. It will apply certain UAX#9 rules
// immediately to relief the parser.
//
// TODO Implement W1 on scanner level
//
func (sc *Scanner) bidic(b []byte) (bidi.Class, int) {
	r, sz := utf8.DecodeRune(b)
	if sz > 0 {
		if sc.hasMode(optionTesting) && unicode.IsUpper(r) {
			return bidi.R, sz // during testing, UPPERCASE is R2L
		}
		props, sz := bidi.Lookup(b)
		clz := props.Class()
		sc.setIfStrong(clz)
		switch clz { // do some pre-processing
		case bidi.NSM: // rule W1, handle accents
			switch sc.currClz {
			case bidi.LRI:
				return bidi.L, sz
			case bidi.RLI:
				return bidi.R, sz
			case bidi.PDI:
				return bidi.ON, sz
			}
			return sc.currClz, sz
		case bidi.EN: // rule W2 and pretext to W7
			if sc.currClz == bidi.L {
				return bidi.L, sz
			}
			switch sc.strong {
			case bidi.AL:
				return bidi.AN, sz
			case bidi.L:
				return LEN, sz
			}
		case bidi.S:
			fallthrough
		case bidi.WS:
			return NI, sz
		case bidi.ON:
			if props.IsBracket() { // rule BD16
				//T().Debugf("Bracket detected: %c", r)
				//T().Debugf("Bracket '%c' with sc.strong = %s", r, ClassString(sc.strong))
				switch sc.strong {
				case bidi.L:
					if props.IsOpeningBracket() {
						return LBRACKO, sz
					}
					return LBRACKC, sz
				case bidi.R:
					if props.IsOpeningBracket() {
						return LBRACKC, sz
					}
					return LBRACKO, sz
				}
				// var isbr bool
				// if isbr, sc.brackets = sc.brackets.pushIfBracket(r); isbr {
				// 	switch sc.strong {
				// 	case bidi.L:
				// 		T().Debugf("- detected an opening bracket")
				// 		return LBRACKO, sz
				// 	case bidi.R:
				// 		return RBRACKC, sz
				// 	}
				// } else if isbr, sc.brackets = sc.brackets.popWith(r); isbr {
				// 	T().Debugf("- detected a closing bracket")
				// 	return BRACKC, sz
				// }
			}
			if sc.currClz == NI {
				return NI, sz
			}
		}
		return props.Class(), sz
	}
	return bidi.L, 0
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

func (sc *Scanner) SetErrorHandler(h func(error)) {
	// TODO
}

// --- Bidi_Class ------------------------------------------------------------

// We use some additional Bidi_Classes, which reflects additional knowledge about
// a character. Our scanner will process some Bidi rules before the parser is
// going to see the tokens.
const (
	LEN     bidi.Class = iota + 100 // left biased european number (EN)
	LBRACKO                         // opening bracket in L context
	RBRACKO                         // opening bracket in L context
	LBRACKC                         // closing bracket in L context
	RBRACKC                         // closing bracket in R context
	BRACKC                          // closing bracket
	NI                              // neutral character
	ILLEGAL bidi.Class = 999        // in-band value denoting illegal class
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

type bracketStack []BracketPair

func (bs bracketStack) push(b rune) bracketStack {
	for _, pair := range UAX9BracketPairs {
		if pair.o == b {
			return append(bs, pair)
		}
	}
	T().Errorf("Push of %c failed, not found as opening bracket")
	return bs
}

func (bs bracketStack) pushIfBracket(b rune) (bool, bracketStack) {
	props, _ := bidi.LookupRune(b)
	if props.IsBracket() && props.IsOpeningBracket() {
		return true, bs.push(b)
	}
	return false, bs
}

func (bs bracketStack) popWith(b rune) (bool, bracketStack) {
	if len(bs) == 0 {
		return false, bs
	}
	i := len(bs) - 1
	for i >= 0 {
		if bs[i].c == b {
			bs = bs[:i]
			return true, bs
		}
	}
	return false, bs
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
