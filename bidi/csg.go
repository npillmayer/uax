package bidi

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/npillmayer/gorgo/lr/scanner"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/bidi/trie"
	"golang.org/x/text/unicode/bidi"
)

type bidiRule struct {
	name   string     // name of the rule according to UAX#9
	lhsLen int        // number of symbols in the left hand side (LHS)
	pass   int        // this is a 2-pass system
	action ruleAction // action to perform on match of LHS
}

// ruleAction is an action on bidi class intervals. Input is a slice of (consecutive)
// class intervals which have been matched. The action's task is to substitute all or some
// of the input intervals by one or more output intervals (reduce action). The ``cursor''
// will be positioned after the substitution by the parser, according to the second result
// of the action, an integer. This position hint will be negative most of the time, telling
// the parser to backtrack and try to re-apply other BiDi rules.
type ruleAction func([]intv) ([]intv, int)

// --- Parser ----------------------------------------------------------------

// TODO Parser is probably not a good naming in this context. We should stick closer
// to UAX#9 (and--insofar it makes sense--to objects in the unicode.bidi package).
// UAX#9 mentions parsing as well, but we should prefer a different word.

// Parse accepts character input and returns a BiDi ordering for the characters.
// inp should be the text of a paragraph, but this is not enforced.
func Parse(inp io.Reader, opts ...Option) *Ordering {
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
	sc    *bidiScanner
	stack []intv
	sp    int // 'pointer' into the stack; start of LHS matching
	trie  *trie.TinyHashTrie
}

// newParser creates a Parser, which is used to parse paragraphs of text and identify
// bidi runs. Parser is made public for cases where clients want to provide their own
// implementation of a scanner. Usually it's much simpler to call bidi.Parse(…)
func newParser(sc *bidiScanner) (*parser, error) {
	p := &parser{
		sc:    sc,
		stack: make([]intv, 0, 64),
		trie:  prepareRulesTrie(),
		sp:    0,
	}
	if p.trie == nil {
		return nil, errors.New("internal error creating rules table")
	}
	return p, nil // TODO check sc
}

func (p *parser) reduce(n int, rhs []intv) {
	T().Debugf("REDUCE at %d: %d⇒%v", p.sp, n, rhs)
	diff := len(rhs) - n
	for i, iv := range rhs {
		p.stack[p.sp+i] = iv
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

// pass1 scans the complete input (character-)sequence, creating an intervals for each
// cluster of characters. Then we do an immediate match for pass-1 rules, which are
// basically the Wx-rules from section 3.3.4 “Resolving Weak Types”.
//
func (p *parser) pass1() {
	la := 0              // length of lookahead LA
	t, _ := p.read(3, 0) // initially load 3 bidi clusters
	var rule, shortrule *bidiRule
	walk := false // if true, accept walking over 1 single cluster
	for {         // scan the complete input sequence (until EOF)
		la = len(p.stack) - p.sp
		t, k := p.read(3-la, t) // extend LA to |LA|=3, if possible
		la += k
		//T().Debugf("t=%v, sp=%d, la=%d, walk=%v", t, p.sp, la, walk) //, minMatchLen)
		if la == 0 {
			if t != scanner.EOF { // TODO remove this
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

// read reads k ≤ n bidi clusters from the scanner. If k < n, EOF has been encountered.
// Returns k.
func (p *parser) read(n int, t int) (int, int) {
	if n <= 0 || t == scanner.EOF {
		return t, 0
	}
	i := 0
	for ; i < n; i++ { // read n bidi clusters
		var pos, length uint64
		var strong interface{}
		t, strong, pos, length = p.sc.NextToken(nil)
		if t == scanner.EOF {
			break
		}
		iv := intv{l: pos, r: pos + length, clz: bidi.Class(t), strong: strong.(strongDist)}
		p.stack = append(p.stack, iv)
	}
	return t, i
}

func (p *parser) pass2() {
	p.sp = 0
	for p.sp < len(p.stack) {
		e := min(len(p.stack), p.sp+3)
		T().Debugf("trying to match %v at %d", p.stack[p.sp:e], p.sp)
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

// Ordering starts the parse and returns a bidi-ordering for the input-text given
// when creating the parser.
func (p *parser) Ordering() *Ordering {
	T().Debugf("--- pass 1 ---")
	p.pass1()
	T().Debugf("--------------")
	T().Debugf("STACK = %v", p.stack)
	T().Debugf("--- pass 2 ---")
	p.pass2()
	T().Debugf("--------------")
	return &Ordering{intervals: p.stack}
}

// matchRulesLHS trys to find 2 rules matching a given interval:
// a long one (returned as the first return value), and possibly one just matching
// the first interval (returned as the second return value).
//
// If either of the two is shorter than minlen, it is not returned. That may
// result in only the long rule being returned.
//
func (p *parser) matchRulesLHS(intervals []intv, minlen int) (*bidiRule, *bidiRule) {
	//T().Debugf("match: %v", intervals)
	iterator := p.trie.Iterator()
	var pointer, entry, short int
	for k, iv := range intervals {
		pointer = iterator.Next(int8(iv.clz))
		//T().Debugf(" pointer[%d]=%d", iv.clz, pointer)
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

// --- Ordering --------------------------------------------------------------

// An Ordering holds the computed visual order of bidi-runs of a paragraph of text.
type Ordering struct {
	// TODO
	// intervals = runs ?
	intervals []intv
}

func (o *Ordering) String() string {
	var b strings.Builder
	for _, iv := range o.intervals {
		b.WriteString(fmt.Sprintf("[%d-%s-%d] ", iv.l, ClassString(iv.clz), iv.r))
	}
	return b.String()
}

// ---------------------------------------------------------------------------

type intv struct {
	l, r   uint64     // left and right bounds, r not included
	clz    bidi.Class // bidi character class of this interval
	strong strongDist // positions of last strong bidi characters
	child  *intv      // some intervals may have a child worth saving
}

func (iv intv) clone() *intv {
	return &intv{
		l:     iv.l,
		r:     iv.r,
		clz:   iv.clz,
		child: iv.child,
	}
}
func (iv intv) String() string {
	return fmt.Sprintf("[%d-%s-%d] ", iv.l, ClassString(iv.clz), iv.r)
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
