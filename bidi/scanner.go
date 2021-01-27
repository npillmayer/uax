package bidi

// Notes:
// - Input reader will in most cases be a cord reader
// - implement Implicit Directional Formatting Characters 	LRM, RLM, ALM
// - If a IRS turns out to have no matching PDI and should be treated as NI,
//   nestend IRS levels must be unified and stored bracket pairs must be
//   corrected. parser will stop IRS after pass 1 and return false
//   Remark: I will not implement this as I consider it a waste of time
// - B Paragraph Separator   PARAGRAPH SEPARATOR, appropriate Newline Functions,
//                           higher-level protocol paragraph determination
// - TODO remove several safety panics, as soon as package is stable

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/bidi"
)

// bidiScanner will read runs of text as a unit, as long as all runes therein have the
// same Bidi class.
type bidiScanner struct {
	mode          uint8                           // scanner modes, set by scanner options
	runeScanner   *bufio.Scanner                  // we're using an embedded rune reader
	bd16          *bracketPairHandler             // support type for handling bracket pairings
	markup        OutOfLineBidiMarkup             // markup for isolating run sequence delimiting
	lastMarkupPos int64                           // last position of out-of-line markup
	lastMarkup    int                             // last / remaining markup
	IRS           map[charpos]*bracketPairHandler // isolating run sequences and their pair handlers
	IRSStack      []charpos                       // start positions of IRSs
	ctxstack      []dirContext                    // context stack for IRSs
}

// NewScanner creates a scanner for bidi formatting. It will read runs of text
// as a unit, as long as all runes therein have the same Bidi_Class.
//
// Clients will provide a Reader and zero or more scanner options. Runes will be
// read from the reader and possibly concatenated to chunks of text with
// identical BiDi class (see NextToken).
func newScanner(input io.Reader, markup OutOfLineBidiMarkup, opts ...Option) *bidiScanner {
	sc := &bidiScanner{}
	sc.runeScanner = bufio.NewScanner(input)
	sc.runeScanner.Split(bufio.ScanRunes)
	sc.markup = markup
	sc.lastMarkup = -1
	sc.bd16 = makeBracketPairHandler(0, nil)
	sc.IRS = make(map[charpos]*bracketPairHandler)
	sc.IRS[0] = sc.bd16
	sc.IRSStack = make([]charpos, 0, 16)
	sc.IRSStack = append(sc.IRSStack, 0)
	for _, opt := range opts {
		if opt != nil {
			opt(sc)
		}
	}
	return sc
}

func markupToBidi(m int) bidi.Class {
	switch m {
	case MarkupLRI, MarkupPDILRI:
		return bidi.LRI
	case MarkupRLI, MarkupPDIRLI:
		return bidi.RLI
	case MarkupPDI:
		return bidi.PDI
	}
	return 0
}

// nextRune reads the next rune from the input reader.
// Returns the rune, its byte length, bidi class and a flag indicating
// a valid input (false for EOF).
func (sc *bidiScanner) nextRune(pos charpos) (rune, int, bidi.Class, bool) {
	if sc.markup != nil && sc.lastMarkup > 0 { // we have a left-over markup to handle
		sc.lastMarkupPos = int64(pos)
		last := sc.lastMarkup
		sc.lastMarkup = 0
		return 0, 0, markupToBidi(last), true
	} else if sc.markup != nil && sc.lastMarkupPos < int64(pos) {
		T().Debugf("bidi scanner: checking for markup at position %d", pos)
		if d := sc.markup(uint64(pos)); int(d) > 0 {
			sc.lastMarkupPos = int64(pos)
			if d&MarkupPDI > 0 { // always handle PDI first
				sc.lastMarkup = d - MarkupPDI // put other markup half on shelf, if any
				return 0, 0, bidi.PDI, true
			}
			sc.lastMarkup = 0
			return 0, 0, markupToBidi(d), true
		}
	}
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
	if bidiclz == bidi.B { // paragraph separator read
		if !sc.hasMode(optionIgnoreParSep) {
			return 0, 0, cNULL, false // treat is as EOF
		}
		bidiclz = bidi.WS // convert it to whitespace
	}
	if props.IsBracket() {
		if props.IsOpeningBracket() {
			bidiclz = cBRACKO
		} else {
			bidiclz = cBRACKC
		}
	}
	T().Debugf("bidi scanner rune %#U (%s)", r, classString(bidiclz))
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
// uint32.
//
func (sc *bidiScanner) Scan(pipe chan<- scrap) {
	var lookahead scrap
	current := sc.initialOuterScrap(false)
	var lastAL charpos // position of last AL character-run in input
	for {
		r, length, bidiclz, ok := sc.nextRune(current.r) // read the next input rune
		if !ok {                                         // if EOF, drain lookahead and quit
			if isisolate(current) { // if last character is PDI
				sc.handleIsolatingRunSwitch(current)
			}
			sc.post(current, pipe) // use lookahead as current scrap and put it onto pipe
			sc.stop(pipe)          // then send quit signal to parser
			break
		}

		// a rune has successfully been read ⇒ make it the new lookahead
		if sc.hasMode(optionTesting) {
			if unicode.IsUpper(r) {
				bidiclz = bidi.R // during testing, UPPERCASE is R2L
			} else { // check for isolating run sequence delimiters
				bidiclz = setTestingIRSDelimiter(r, bidiclz)
			}
		}
		isAL := bidiclz == bidi.AL                     // AL will be changed by rule W3
		bidiclz = applyRulesW1to3(r, bidiclz, current) // UAX#9 W1–3 handled by scanner
		//lookahead = makeScrap(r, bidiclz, lapos, length)
		lookahead = makeScrap(r, bidiclz, current.r, length)
		T().Debugf("bidi scanner lookahead = %v", lookahead) // finally a new lookahead

		// the current scrap is finished if the lookahead cannot extend it
		if current.bidiclz == cNULL || current.bidiclz != lookahead.bidiclz || isbracket(current) || isisolate(current) {
			if current.bidiclz != cNULL { // if the current scrap is not the initial null scrap
				if isisolate(current) {
					sc.handleIsolatingRunSwitch(current)
				}
				sc.post(current, pipe) // put current on channel
			}
			// proceed ahead, making lookahead the current scrap
			if isisolate(lookahead) && lookahead.bidiclz != bidi.PDI {
				sc.ctxstack = append(sc.ctxstack, current.context)
			} else if current.bidiclz == bidi.PDI && len(sc.ctxstack) > 0 {
				current.context = sc.ctxstack[len(sc.ctxstack)-1]
				sc.ctxstack = sc.ctxstack[:len(sc.ctxstack)-1]
			}
			lookahead = inheritStrongTypes(lookahead, current, lastAL)
			current = sc.prepareRuleBD16(r, lookahead)
			if isAL {
				lastAL = current.l
			}
		} else { // otherwise the current scrap grows
			current = collapse(current, lookahead, current.bidiclz) // meld LA with current
			T().Debugf("current bidi scanner scrap = %s, next iteration", current)
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

func (sc *bidiScanner) initialOuterScrap(setIRS bool) scrap {
	var current scrap
	current.bidiclz = cNULL
	if sc.hasMode(optionOuterR2L) {
		T().Infof("resolving paragraph with R2L embedding context")
		current.context.SetEmbedding(bidi.RightToLeft)
		if setIRS {
			current.bidiclz = bidi.RLI
		}
	} else {
		T().Infof("resolving paragraph with L2R embedding context")
		current.context.SetEmbedding(bidi.LeftToRight)
		if setIRS {
			current.bidiclz = bidi.LRI
		}
	}
	return current
}

// applyRulesW1to3 returns the Bidi_Class for a rune. It will apply certain UAX#9
// rules immediately to relief the parser.
//
func applyRulesW1to3(r rune, clz bidi.Class, current scrap) bidi.Class {
	currclz := current.bidiclz
	switch clz { // do some pre-processing
	case bidi.NSM: // rule W1, handle accents
		switch currclz {
		case bidi.LRI:
			return bidi.L
		case bidi.RLI:
			return bidi.R
		case bidi.PDI:
			return cNI
		}
		return currclz
	case bidi.EN: // rule W2
		if current.context.IsAL() {
			return bidi.AN
		}
	case bidi.AL: // rule W3
		return bidi.R
	case bidi.S, bidi.WS, bidi.ON:
		return cNI
		//if sc.bidiclz == NI {
		// return NI
		//}
	}
	return clz
}

func (sc *bidiScanner) prepareRuleBD16(r rune, s scrap) scrap {
	if !isbracket(s) {
		return s
	}
	if s.bidiclz == cBRACKO {
		// is LA not just a bracket, but part of a UAX#9 bracket pair?
		isbr := sc.bd16.pushOpening(r, s)
		if isbr {
			T().Debugf("bidi scanner pushed lookahead onto bracket stack: %s", s)
			sc.bd16.dump()
		}
	} else {
		found, _ := sc.bd16.findPair(r, s)
		if found {
			T().Debugf("bidi scanner popped closing bracket: %s", s)
			sc.bd16.dump()
		}
	}
	return s
}

// isAL is true if dest has been of bidi class AL (before UAX#9 rule W3 changed it)
func inheritStrongTypes(dest, src scrap, lastAL charpos) scrap {
	T().Debugf("bidi scanner: inherit %s => %s     %v", src, dest, src.context)
	dest.context = src.context
	switch dest.bidiclz {
	case bidi.LRI:
		dest.context.embeddingDir = bidi.L
	case bidi.RLI:
		dest.context.embeddingDir = bidi.R
		//panic("R2")
	default:
		dest.context.SetStrongType(bidi.AL, lastAL)
		switch src.bidiclz {
		case bidi.L, bidi.LRI:
			dest.context.SetStrongType(bidi.L, src.l)
			T().Debugf("bidi scanner LA has L context=%v from %v", dest.context, src.context)
		case bidi.R, bidi.RLI:
			dest.context.SetStrongType(bidi.R, src.l)
			T().Debugf("bidi scanner LA has R context=%v from %v", dest.context, src.context)
		case bidi.AL:
			dest.context.SetStrongType(bidi.AL, src.l)
			T().Debugf("la has AL context=%v from %v", dest.context, src.context)
		}
	}
	return dest
}

// --- Nesting isolating run sequences ---------------------------------------

func isisolate(s scrap) bool {
	switch s.bidiclz {
	case bidi.LRI, bidi.RLI, bidi.FSI, bidi.PDI:
		return true
	}
	return false
}

func (sc *bidiScanner) handleIsolatingRunSwitch(s scrap) {
	if s.bidiclz == bidi.PDI {
		// re-establish outer BD16 handler
		if len(sc.IRSStack) == 0 {
			T().Debugf("bidi scanner found non-paired PDI at position %d", s.l)
			return
		}
		sc.bd16.lastpos = s.l
		sc.IRSStack = sc.IRSStack[:len(sc.IRSStack)-1] // pop current IRS level
		if len(sc.IRSStack) > 0 {
			tos := sc.IRSStack[len(sc.IRSStack)-1]
			sc.bd16 = sc.IRS[tos]
		}
		T().Debugf("bidi scanner read PDI, switch back to outer IRS at %d", sc.bd16.firstpos)
		return
	}
	T().Debugf("bidi scanner handleIsolatingRunSwitch(%v | %v)", s, s.context)
	// establish new BD16 handler
	irs := sc.IRS[0]
	for irs.next != nil { // find most rightward isolating run sequence
		irs = irs.next
	}
	sc.bd16 = makeBracketPairHandler(s.l, irs)
	sc.IRS[s.l] = sc.bd16
	sc.IRSStack = append(sc.IRSStack, s.l)
	T().Debugf("bidi scanner: new IRS with position %d, nesting level is %d", sc.bd16.firstpos, len(sc.IRSStack)-1)
}

func (sc *bidiScanner) findBD16ForPos(pos charpos) *bracketPairHandler {
	irs := sc.IRS[0]
	bd16 := irs
	for irs.next != nil { // find most rightward isolating run sequence
		if irs.firstpos <= pos {
			if irs.lastpos == 0 || irs.lastpos >= pos {
				bd16 = irs
			}
		}
		irs = irs.next
	}
	if bd16 == nil {
		panic(fmt.Sprintf("bidi scanner: could not find IRS for position %d", pos))
	}
	return bd16
}

// As explained for option TestingMode(…), certain characters are interpreted as
// isolating run sequence delimiters during testing.
func setTestingIRSDelimiter(r rune, clz bidi.Class) bidi.Class {
	switch r {
	case '>':
		return bidi.LRI
	case '<':
		return bidi.RLI
	case '=':
		return bidi.PDI
	}
	return clz
}

// --- Out-of-line markup ----------------------------------------------------

// OutOfLineBidiMarkup is queried during read of input text for out-of-line
// Bidi delimiters (LRI, RLI, PDI). Such markup may result, e.g., from HTML attributes
// or CSS styles. It receives a text position and—if appropriate—returns a
// Bidi class to be inserted. It will be treated by the resolver as a Bidi delimiter
// of byte-length zero.
type OutOfLineBidiMarkup func(uint64) int

// Constants to use by clients as OutOfLineBidiMarkup return values.
const (
	MarkupLRI    int = int(bidi.LRI) << 8
	MarkupRLI    int = int(bidi.RLI) << 8
	MarkupPDI    int = int(bidi.PDI)
	MarkupPDILRI int = MarkupPDI | MarkupLRI
	MarkupPDIRLI int = MarkupPDI | MarkupRLI
)

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

// Option configures a Bidi algorithm
type Option func(p *bidiScanner)

const (
	optionRecognizeLegacy uint8 = 1 << 1 // recognize LRM, RLM, ALM, LRE, RLE, LRO, RLO, PDF
	optionOuterR2L        uint8 = 1 << 2 // set outer direction as RtoL
	optionTesting         uint8 = 1 << 3 // test mode: recognize uppercase as class R
	optionIgnoreParSep    uint8 = 1 << 4 // interpret paragraph separators as whitespace
)

// RecognizeLegacy is not yet implemented. It was indented to make the
// resolver recognize legacy formatting, i.e.
// LRE, RLE, LRO, RLO, PDF. However, I changed my mind and
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
// Additionally we follow a convention of the UAX#9 algorithm documentation:
// “The invisible, zero-width formatting characters LRI, RLI, and PDI are
// represented with the symbols '>', '<', and '=', respectively.” Thus it is
// possible to replay the examples of section 3.4 of UAX#9:
//
//     <car MEANS CAR.=
//
// or
//
//     DID YOU SAY ’>he said “<car MEANS CAR=”=‘?
//
func TestMode(b bool) Option {
	return func(sc *bidiScanner) {
		if !sc.hasMode(optionTesting) && b ||
			sc.hasMode(optionTesting) && !b {
			sc.mode |= optionTesting
		}
	}
}

// IgnoreParagraphSeparators determines wether paragraph separators (i.e., newlines
// at al.) are to be ignored and interpretet as whitespace instead.
// The default value for this option is `false`, resulting in effectively
// interpreting any paragraph separator as `end of input`. In this case the
// paragraph separator is cut off of the input.
func IgnoreParagraphSeparators(b bool) Option {
	return func(sc *bidiScanner) {
		if !sc.hasMode(optionIgnoreParSep) && b ||
			sc.hasMode(optionIgnoreParSep) && !b {
			sc.mode |= optionIgnoreParSep
		}
	}
}

func (sc *bidiScanner) hasMode(m uint8) bool {
	return sc.mode&m > 0
}
