package bidi

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/npillmayer/uax/bidi/trie"
	"golang.org/x/text/unicode/bidi"
)

// --- Parser / Resolver -----------------------------------------------------

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
	stack []scrap            // we manage a stack of bidi class scraps
	sp    int                // 'pointer' into the stack; start of LHS matching
	trie  *trie.TinyHashTrie // dictionary of bidi rules
}

// newParser creates a Parser, which is used to parse paragraphs of text and identify
// bidi runs. Parser is made public for cases where clients want to provide their own
// implementation of a scanner. Usually it's much simpler to call bidi.Parse(…)
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

func (p *parser) reduce(n int, rhs []scrap) {
	T().Debugf("REDUCE at %d: %d⇒%v", p.sp, n, rhs)
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
	la := 0              // length of lookahead LA
	t, _ := p.read(3, 0) // initially load 3 scraps
	var rule, shortrule *bidiRule
	walk := false // if true, accept walking over 1 scrap
	for {         // scan the complete input sequence (until EOF)
		la = len(p.stack) - p.sp
		t, k := p.read(3-la, t) // extend LA to |LA|=3, if possible
		la += k
		//T().Debugf("t=%v, sp=%d, la=%d, walk=%v", t, p.sp, la, walk) //, minMatchLen)
		if la == 0 {
			if t != NULL { // TODO remove this
				panic("no LA, but not at EOF?")
			}
			break
		}
		T().Debugf("trying to match %v at %d", p.stack[p.sp:len(p.stack)], p.sp)
		if walk {
			rule = shortrule
			if rule == nil || rule.pass > 1 {
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
		rhs, jmp := rule.action(p.stack[p.sp:len(p.stack)])
		p.reduce(rule.lhsLen, rhs)
		p.sp = max(0, p.sp+jmp) // avoid jumping left of 0
		walk = false
		//T().Debugf("next iteration, reading token")
	}
}

// nextInputScrap reads the next scrap from the scanner pipe. It returns a
// new scrap and false if this is the EOF scrap, true otherwise.
func (p *parser) nextInputScrap(pipe <-chan scrap) (scrap, bool) {
	s := <-pipe
	if s.bidiclz == NULL {
		return s, false
	}
	return s, true
}

// read reads k ≤ n bidi clusters from the scanner. If k < n, EOF has been encountered.
// Returns k.
func (p *parser) read(n int, t bidi.Class) (bidi.Class, int) {
	if n <= 0 || t == NULL {
		return t, 0
	}
	i := 0
	for ; i < n; i++ { // read n bidi clusters
		// var pos, length, r uint64
		// var strong interface{}
		// t, strong, pos, length = p.sc.NextToken(nil)
		s, ok := p.nextInputScrap(p.pipe)
		if !ok {
			// TODO set EOF
		}
		t = s.bidiclz
		if t == NULL {
			break
		}
		// s := scrap{l: pos, bidiclz: bidi.Class(t), strong: strong.(strongTypes)}
		// if s.bidiclz == BRACKC { // closing brackets have misused length field
		// 	r = length
		// } else {
		// 	r = pos + length
		// }
		// s.r = r
		p.stack = append(p.stack, s)
	}
	return t, i
}

func (p *parser) pass2() {
	p.sp = 0
	for p.sp < len(p.stack) {
		e := min(len(p.stack), p.sp+3)
		T().Debugf("trying to match %v at %d", p.stack[p.sp:e], p.sp)
		if p.stack[p.sp].bidiclz == BRACKC {
			p.performRuleN0()
			p.sp++
			continue
		}
		rule, _ := p.matchRulesLHS(p.stack[p.sp:len(p.stack)], 2)
		if rule == nil || rule.pass < 2 {
			p.sp++
			continue
		}
		T().Debugf("applying UAX#9 rule %s", rule.name)
		rhs, jmp := rule.action(p.stack[p.sp:len(p.stack)])
		p.reduce(rule.lhsLen, rhs)
		p.sp = max(0, p.sp+jmp) // avoid jumping left of 0
	}
}

func (p *parser) performRuleN0() {
	T().Debugf("applying UAX#9 rule N0 (bracket pairs) with %s", p.stack[p.sp])
	obrpos := p.stack[p.sp].r // position of opening bracket
	T().Debugf("position of opening bracket should be %d", obrpos)
	osp := p.findOpeningBracket(obrpos)
	if osp >= 0 {
		// osp is stack index of interval (of length 1) for opening bracket
		p.stack[p.sp].r = 1 // reset the misused r field, no longer needed
		openBr := p.stack[osp]
		closeBr := p.stack[p.sp]
		//ol, or := openBr.strong.LRPos()
		cl, cr := closeBr.strong.LRPos()
		emb := closeBr.strong.EmbeddingDir()
		// a. Inspect the bidirectional types of the characters enclosed within the
		//    bracket pair.
		if charpos(cl) > openBr.l && emb == bidi.L {
			// L in brackets and embedding dir = L
			// b. If any strong type (either L or R) matching the embedding direction
			//    is found, set the type for both brackets in the pair to match the
			//    embedding direction.
		} else if charpos(cr) > openBr.l && emb == bidi.R {
			// R in brackets and embedding dir = R
			// b. If any strong type (either L or R) matching the embedding direction
			//    is found, set the type for both brackets in the pair to match the
			//    embedding direction.
		} else if charpos(cl) > openBr.l && emb == bidi.R {
			// L in brackets and embedding dir = R
			// c. Otherwise, if there is a strong type it must be opposite the embedding
			//    direction. Therefore, test for an established context with a preceding
			//    strong type by checking backwards before the opening paired bracket
			//    until the first strong type (L, R, or sos) is found.
			if openBr.strong.Context() == bidi.L {
				// c.1. If the preceding strong type is also opposite the embedding
				//      direction, context is established, so set the type for both
				//      brackets in the pair to that direction.
			} else {
				// c.2. Otherwise set the type for both brackets in the pair to the
				//      embedding direction.
			}
		} else if charpos(cr) > openBr.l && emb == bidi.L {
			// R in brackets and embedding dir = L
			// c. Otherwise, if there is a strong type it must be opposite the embedding
			//    direction. Therefore, test for an established context with a preceding
			//    strong type by checking backwards before the opening paired bracket
			//    until the first strong type (L, R, or sos) is found.
			if openBr.strong.Context() == bidi.R {
				// c.1. If the preceding strong type is also opposite the embedding
				//      direction, context is established, so set the type for both
				//      brackets in the pair to that direction.
			} else {
				// c.2. Otherwise set the type for both brackets in the pair to the
				//      embedding direction.
			}
		} else {
			// d. Otherwise, there are no strong types within the bracket pair.
			//    Therefore, do not set the type for that bracket pair.
		}
	} else {
		panic("could not find opening bracket")
	}
}

// Ordering starts the parse and returns a bidi-ordering for the input-text gsen
// when creating the parser.
func (p *parser) Ordering() *Ordering {
	T().Debugf("--- pass 1 ---")
	p.pass1()
	T().Debugf("--------------")
	T().Debugf("STACK = %v", p.stack)
	T().Debugf("--- pass 2 ---")
	p.pass2()
	T().Debugf("--------------")
	return &Ordering{scraps: p.stack}
}

// matchRulesLHS trys to find 2 rules matching a gsen interval:
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

func (p *parser) findOpeningBracket(pos charpos) int {
	o := p.sp - 1
	for o >= 0 {
		s := p.stack[o]
		if s.l <= pos && s.r > pos {
			T().Debugf("found interval at position for closing bracket")
			if s.bidiclz != BRACKO {
				panic("interval at bracket position is not a bracket")
			}
			return o
		}
		o--
	}
	return -1 // in-band return value: not found
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
		b.WriteString(fmt.Sprintf("[%d-%s-%d] ", s.l, ClassString(s.bidiclz), s.r))
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
	trie, err := trie.NewTinyHashTrie(103, int8(MAX))
	if err != nil {
		T().Errorf(err.Error())
		panic(err.Error())
	}
	var r *bidiRule
	tracelevel := T().GetTraceLevel()
	//T().SetTraceLevel(tracing.LevelInfo)
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
	T().Infof("#categories=%d", MAX)
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
