package bidi

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/bidi/trie"
)

// --- Parser / Resolver -----------------------------------------------------

// We construct a parser which processes grammar rules for a context sensitive
// grammar. These rules will resemble the rules of UAX#9 as closely as possible.
// Please note that this is by no means a general purpose parser. It is tightly
// coupled to the bidi rules and their characteristics.
//
// The parser is a 2-pass system: the first pass reads in characters of a
// paragraph and converts runs of characters to clusters of bidi classes, which
// we will call 'scraps'. During the read it applies rules W* of UAX#9.
// The second pass iterates of existing scraps left on a stack by pass 1 and
// applies rules N* of UAX#9.

// ResolveParagraph accepts character input and returns a BiDi ordering for the characters.
// inp should be the text of a single paragraph, but this is not enforced.
//
// UAX#9 lists the following phases for bidi typesetting:
//    3.3  Resolving Embedding Levels
//    3.4  Reordering Resolved Levels
//    3.5  Shaping
// Resolving means identifying runs of left-to-right or right-to-left text fragements.
//
// The subsequent phases (3.4 and 3.5) require the text to be segmented into lines,
// which is not handled by this package.
//
func ResolveParagraph(inp io.Reader, opts ...Option) *Ordering {
	sc := newScanner(inp, opts...)
	p, err := newParser(sc) // TODO create a global one and re-use it
	if err != nil {
		panic(fmt.Sprintf("something went wrong creating a parser: %s", err.Error()))
	}
	return p.Ordering()
}

// parser is used to parse paragraphs of text and identify bidi runs.
// clients will not instantiate one, but rather call bidi.Parse(…)
type parser struct {
	sc    *bidiScanner       // parser uses a bidi-specific scanner
	pipe  chan scrap         // communication with the scanner
	eof   bool               // end of input reached during first pass
	stack []scrap            // we manage a stack of bidi class scraps
	sp    int                // 'pointer' into the stack; start of LHS matching
	trie  *trie.TinyHashTrie // dictionary of bidi rules
}

// newParser creates a Parser, which is used to parse paragraphs of text and identify
// bidi runs.
func newParser(sc *bidiScanner) (*parser, error) {
	p := &parser{
		sc:    sc,
		stack: make([]scrap, 0, 64),
		trie:  prepareRulesTrie(),
		sp:    0,
	}
	if p.trie == nil {
		return nil, errors.New("internal error creating rules table")
	}
	return p, nil // TODO check sc
}

// reduce applies a rule to the scraps on the stack. It takes n scraps, which need
// not necessarily be the top scraps, and replaces them with the right-hand-side (RHS)
// of the applied rule.
func (p *parser) reduce(n int, rhs []scrap) {
	T().Debugf("REDUCE at %d: %d ⇒ %v", p.sp, n, rhs)
	diff := len(rhs) - n
	for i, s := range rhs {
		p.stack[p.sp+i] = s
	}
	//pos := max(0, len(p.stack)-n+len(rhs)) // avoid jumping left of 0
	//pos := max(0, len(p.stack)+diff)
	pos := max(0, p.sp+len(rhs))
	//T().Debugf("STACK = %v", p.stack)
	//T().Debugf("sp = %d, diff = %d, pos = %d", p.sp, diff, pos)
	p.stack = append(p.stack[:pos], p.stack[pos-diff:]...)
	//T().Debugf("STACK = %v", p.stack)
	T().Debugf("sp=%d, stack-LA is now %v", p.sp, p.stack[p.sp:])
}

// pass1 scans the complete input (character-)sequence, creating an scraps for each
// cluster of characters. Then we do an immediate match for pass-1 rules, which are
// basically the Wx-rules from section 3.3.4 “Resolving Weak Types”.
//
func (p *parser) pass1() {
	la := 0            // length of lookahead LA
	_, ok := p.read(3) // initially load 3 scraps
	if !ok {
		return // no input to read
	}
	var rule, shortrule *bidiRule
	walk := false // if true, accept walking over 1 scrap
	for {         // scan the complete input sequence (until EOF)
		T().Debugf("EOF=%v", p.eof)
		la = len(p.stack) - p.sp
		k, _ := p.read(3 - la) // extend LA to |LA|=3, if possible
		la += k
		//T().Debugf("t=%v, sp=%d, la=%d, walk=%v", t, p.sp, la, walk) //, minMatchLen)
		if la == 0 {
			if !p.eof {
				panic("no LA, but not at EOF?")
			}
			break
		}
		T().Debugf("trying to match %v at %d", p.stack[p.sp:len(p.stack)], p.sp)
		if walk {
			rule = shortrule
			if rule == nil || rule.pass > 1 {
				T().Debugf("walking over %s", p.stack[p.sp])
				p.sp++ // walk by skipping
				walk = false
				continue
			}
		} else { // apply long rule, if present
			rule, shortrule = p.matchRulesLHS(p.stack[p.sp:len(p.stack)], 0) //minMatchLen)
			if rule == nil || rule.pass > 1 {
				walk = true // try matching single bidi cluster
				continue
			}
		}
		T().Debugf("applying UAX#9 rule %s", rule.name)
		rhs, jmp, newL := rule.action(p.stack[p.sp:len(p.stack)])
		p.reduce(rule.lhsLen, rhs)
		if newL {
			bd16 := p.sc.findBD16ForPos(p.stack[p.sp].l)
			bd16.UpdateClosingBrackets(p.stack[p.sp])
		}
		p.sp = max(0, p.sp+jmp) // avoid jumping left of 0
		walk = false
	}
}

// nextInputScrap reads the next scrap from the scanner pipe. It returns a
// new scrap and false if this is the EOF scrap, true otherwise.
func (p *parser) nextInputScrap(pipe <-chan scrap) (scrap, bool) {
	T().Debugf("==> reading from pipe")
	s := <-pipe
	T().Debugf("    read %s from pipe", s)
	if s.bidiclz == cNULL {
		return s, false
	}
	return s, true
}

// read reads k ≤ n bidi clusters from the scanner. If k < n, EOF has been encountered.
// Returns k.
func (p *parser) read(n int) (int, bool) {
	T().Debugf("----> read(%d)", n)
	if n <= 0 || p.eof {
		return 0, false
	}
	i := 0
	for ; i < n; i++ { // read n bidi clusters
		s, ok := p.nextInputScrap(p.pipe)
		if !ok {
			p.eof = true
			break
		}
		p.stack = append(p.stack, s)
	}
	T().Debugf("have read %d scraps, stack=%v", i, p.stack)
	return i, true
}

// pass 2 operates on the scraps laying on the stack, starting at the
// bottom of the stack. The scanner already has stopped and we no longer
// have access to the original text input, but rather must have accumulated
// enough information to perform the Bidi rules of pass 2.
//
// The number of iterations in pass 2 will almost always be much lower
// than for pass 1. Pass 2 will result in a stack on which unresolvable
// scraps remain. These are then the directional runs of an isolating run
// sequence. If an isolating run sequence has just, say, left-to-right text,
// there should be only a single scrap on the stack with bidi class L.
//
func (p *parser) pass2() {
	p.sp = 0
	for p.sp < len(p.stack) {
		e := min(len(p.stack), p.sp+3)
		T().Debugf("trying to match %v at %d", p.stack[p.sp:e], p.sp)
		//if p.stack[p.sp].bidiclz == cBRACKC {
		if isbracket(p.stack[p.sp]) {
			jmp := p.performRuleN0()
			p.sp = max(0, p.sp+jmp) // avoid jumping left of 0
			continue
		}
		rule, _ := p.matchRulesLHS(p.stack[p.sp:len(p.stack)], 2)
		if rule == nil || rule.pass < 2 {
			p.sp++
			continue
		}
		T().Debugf("applying UAX#9 rule %s", rule.name)
		rhs, jmp, _ := rule.action(p.stack[p.sp:len(p.stack)])
		p.reduce(rule.lhsLen, rhs)
		p.sp = max(0, p.sp+jmp) // avoid jumping left of 0
	}
}

// N0. Process bracket pairs in an isolating run sequence sequentially in the logical
//     order of the text positions of the opening paired brackets using the logic
//     given below. Within this scope, bidirectional types EN and AN are treated as R.
//
func (p *parser) performRuleN0() (jmp int) {
	T().Debugf("applying UAX#9 rule N0 (bracket pairs) with %s", p.stack[p.sp])
	jmp = 1 // default is to walk over the bracket
	if p.stack[p.sp].bidiclz == cBRACKO {
		// Identify the bracket pairs in the current isolating run sequence according to BD16.
		openBr := p.stack[p.sp]
		closeBr, found := p.findCorrespondingBracket(openBr)
		if !found {
			T().Debugf("Did not find closing bracket for %s", openBr)
			closeBr.bidiclz = cNI
			return
		}
		T().Debugf("closing bracket for %s is %s", openBr, closeBr)
		T().Debugf("closing bracket has context=%v", closeBr.context)
		T().Debugf("closing bracket has match pos=%d", closeBr.context.matchPos)
		// a. Inspect the bidirectional types of the characters enclosed within the
		//    bracket pair.
		if closeBr.HasEmbeddingMatchAfter(openBr) {
			// b. If any strong type (either L or R) matching the embedding direction
			//    is found, set the type for both brackets in the pair to match the
			//    embedding direction.
			openBr.bidiclz = openBr.context.embeddingDir
			jmp = -2
		} else if closeBr.HasOppositeAfter(openBr) {
			// c. Otherwise, if there is a strong type it must be opposite the embedding
			//    direction. Therefore, test for an established context with a preceding
			//    strong type by checking backwards before the opening paired bracket
			//    until the first strong type (L, R, or sos) is found.
			if openBr.StrongContext() == openBr.o() {
				// c.1. If the preceding strong type is also opposite the embedding
				//      direction, context is established, so set the type for both
				//      brackets in the pair to that direction.
				openBr.bidiclz = opposite(openBr.context.embeddingDir)
			} else {
				// c.2. Otherwise set the type for both brackets in the pair to the
				//      embedding direction.
				openBr.bidiclz = openBr.context.embeddingDir
			}
			jmp = -2
		} else {
			T().Debugf("no strong types found within bracket pair")
			// d. Otherwise, there are no strong types within the bracket pair.
			//    Therefore, do not set the type for that bracket pair.
			openBr.bidiclz = cNI
			jmp = -1
		}
		p.changeBracketBidiClass(openBr)
		p.stack[p.sp] = openBr
	} else {
		closeBr := p.stack[p.sp]
		if openBr, found := p.findCorrespondingBracket(closeBr); found {
			closeBr.bidiclz = openBr.bidiclz
		} else {
			closeBr.bidiclz = cNI
		}
		p.stack[p.sp] = closeBr
		if closeBr.bidiclz != cNI {
			jmp = -2
		}
	}
	return
}

// Ordering starts the parse and returns a bidi-ordering for the input-text gsen
// when creating the parser.
func (p *parser) Ordering() *Ordering {
	p.pipe = make(chan scrap, 0)
	go p.sc.Scan(p.pipe)
	T().Debugf("------ pass 1 ------")
	p.pass1()
	T().Debugf("--------------------")
	T().Debugf("STACK = %v", p.stack)
	T().Debugf("------ pass 2 ------")
	p.pass2()
	T().Debugf("--------------------")
	return &Ordering{scraps: p.stack}
}

// matchRulesLHS will match a sequence of scraps laying on the stack to left hand
// sides of grammar rules. In parsing theory, this process is usually called
// handle recgonition and is applied to the top symbols of the parsing stack
// (in reverse sequence). Which handles to look for is determined by a regular
// automaton to restrict the set of possible handles to the legal ones at this
// position of a parse. However, we do not deal with a context free grammar here,
// thus we will not employ an automaton and we will not be restricted to the top
// symbols of the stack. Instead, we will walk the stack from bottom to top,
// trying to reach TOS all the while the scanner puts further scraps onto the stack.
// Matching LHS handles is done mid-stack. However, this is of course a white lie,
// as most of the time during pass 1 we will be operating very close to TOS.
//
// Finding applicable rules via their LHS is supported by the way we store the
// rules: A trie lets us find a rule for a LHS very efficiently. Left hand sides
// of UAX#9 rules are short (3 scraps at most) and a trie lets us find a rule
// with at most 3 index accesses.
//
// matchRulesLHS trys to find 2 rules matching a given interval:
// a long one (returned as the first return value), and possibly one just matching
// the first interval (returned as the second return value).
//
// If either of the two is shorter than minlen, it is not returned. That may
// result in only the long rule being returned.
//
func (p *parser) matchRulesLHS(scraps []scrap, minlen int) (*bidiRule, *bidiRule) {
	//T().Debugf("match: %v", scraps)
	iterator := p.trie.Iterator()
	var pointer, entry, short int
	for k, s := range scraps {
		pointer = iterator.Next(int8(s.bidiclz))
		//T().Debugf(" pointer[%d]=%d", s.clz, pointer)
		if pointer == 0 {
			break
		}
		if k == 0 {
			short = pointer
		} else if k+1 >= minlen { // minlen will never be 0
			entry = pointer
		}
	}
	rule, shortrule := rules[entry], rules[short]
	if entry != 0 && rule != nil {
		T().Debugf("FOUND MATCHing  long rule %s for LHS, pass=%d", rule.name, rule.pass)
	}
	if short != 0 && shortrule != nil {
		T().Debugf("FOUND MATCHing short rule %s for LHS, pass=%d", shortrule.name, shortrule.pass)
	}
	if entry == 0 || rule == nil {
		if short == 0 || shortrule == nil {
			return nil, nil
		}
		return nil, shortrule
	} else if short == 0 || shortrule == nil {
		return rule, nil
	}
	return rule, shortrule
}

func (p *parser) findCorrespondingBracket(s scrap) (scrap, bool) {
	bd16 := p.sc.findBD16ForPos(s.l)
	pair, found := bd16.FindBracketPairing(s)
	if found {
		if s.bidiclz == cBRACKO {
			return pair.closing, true
		}
		return pair.opening, true
	}
	return s, false
}

func (p *parser) changeBracketBidiClass(s scrap) {
	bd16 := p.sc.findBD16ForPos(s.l)
	bd16.changeOpeningBracketClass(s)
}

// --- Ordering --------------------------------------------------------------

// An Ordering holds the computed visual order of bidi-runs of a paragraph of text.
type Ordering struct {
	// TODO
	// scraps = runs ?
	scraps []scrap
}

func (o *Ordering) String() string {
	var b strings.Builder
	for _, s := range o.scraps {
		b.WriteString(fmt.Sprintf("[%d-%s-%d] ", s.l, classString(s.bidiclz), s.r))
	}
	return b.String()
}

// --- Rules trie ------------------------------------------------------------

var rules map[int]*bidiRule      // TODO this is probably not the best idea
var rulesTrie *trie.TinyHashTrie // global dictionary for rules
var prepareTrieOnce sync.Once    // all parsers will share a single rules dictionary

func prepareRulesTrie() *trie.TinyHashTrie {
	//prepareTrieOnce.Do(func() {
	if rules == nil {
		rules = make(map[int]*bidiRule)
	}
	trie, err := trie.NewTinyHashTrie(103, int8(cMAX))
	if err != nil {
		T().Errorf(err.Error())
		panic(err.Error())
	}
	var r *bidiRule
	tracelevel := T().GetTraceLevel()
	T().SetTraceLevel(tracing.LevelInfo)
	var lhs []byte
	// --- allocate all the rules ---
	r, lhs = ruleW4_1()
	allocRule(trie, r, lhs)
	r, lhs = ruleW4_2()
	allocRule(trie, r, lhs)
	r, lhs = ruleW4_3()
	allocRule(trie, r, lhs)
	r, lhs = ruleW5_1()
	allocRule(trie, r, lhs)
	r, lhs = ruleW5_2()
	allocRule(trie, r, lhs)
	r, lhs = ruleW6_1()
	allocRule(trie, r, lhs)
	r, lhs = ruleW6_2()
	allocRule(trie, r, lhs)
	r, lhs = ruleW6_3()
	allocRule(trie, r, lhs)
	r, lhs = ruleW7()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_0()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_1()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_2()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_3()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_4()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_5()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_6()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_7()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_8()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_9()
	allocRule(trie, r, lhs)
	r, lhs = ruleN1_10()
	allocRule(trie, r, lhs)
	r, lhs = ruleL()
	allocRule(trie, r, lhs)
	r, lhs = ruleR()
	allocRule(trie, r, lhs)
	// ------------------------------
	trie.Freeze()
	T().SetTraceLevel(tracelevel)
	T().Debugf("--- freeze trie -------------")
	trie.Stats()
	rulesTrie = trie
	//})
	return rulesTrie
}

func allocRule(trie *trie.TinyHashTrie, rule *bidiRule, lhs []byte) {
	T().Debugf("storing rule %s for LHS=%v", rule.name, lhs)
	pointer := trie.AllocPositionForWord(lhs)
	T().Debugf("  -> %d", pointer)
	rules[pointer] = rule
}

// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
