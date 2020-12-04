package bidi

import (
	"github.com/npillmayer/gorgo/lr/sppf"
	"github.com/npillmayer/gorgo/terex"
	"github.com/npillmayer/gorgo/terex/termr"
)

/*
var atomOp *sExprTermR  // for Atom -> ... productions
var opOp *sExprTermR    // for Op -> ... productions
var quoteOp *sExprTermR // for Quote -> ... productions
var seqOp *sExprTermR   // for Sequence -> ... productions
var listOp *sExprTermR  // for List -> ... productions
*/

type bidiTreeOp struct {
	name string
	//
	opname string
	rules  []termr.RewriteRule
	call   func(terex.Element, *terex.Environment) terex.Element
	quote  func(terex.Element, *terex.Environment) terex.Element
}

func (op *bidiTreeOp) AddRule(pattern *terex.GCons, rew *termr.Rewriter) {
}

// makePattern creates a pattern from a TeREx-lang string.
// Returns a TeREx element to be used in a term rewriting rule.
func makePattern(pattern string) *terex.GCons {
	return nil
}

// Rewrite performs termin rewriting für an AST node of the BiDi parse tree.
//
func (op *bidiTreeOp) Rewrite(l *terex.GCons, env *terex.Environment) terex.Element {
	T().Debugf("%s:Op.Rewrite[%s] called, %d rules", op.Name(), l.ListString(), len(op.rules))
	T().Debugf(env.Dump())
	for _, rule := range op.rules {
		T().Infof("match: trying %s %% %s ?", rule.Pattern.ListString(), l.ListString())
		if rule.Pattern.Match(l, env) {
			T().Infof("Op %s has a match", op.Name())
			//T().Debugf("-> pre rewrite: %s", l.ListString())
			v := rule.Rewrite(l, env)
			//T().Debugf("<- post rewrite:")
			v.Dump(T().GetTraceLevel())
			//terex.DumpElement(v)
			T().Infof("Op %s rewrite -> %s", op.Name(), v.String())
			//return rule.Rewrite(l, env)
			return v
		}
	}
	return terex.Elem(l)
}

func (op *bidiTreeOp) Descend(sppf.RuleCtxt) bool {
	return true
}

func (op *bidiTreeOp) Name() string {
	return op.name
}

func (op *bidiTreeOp) Operator() terex.Operator {
	return op
}

// String is part of interface terex.Operator
func (op *bidiTreeOp) String() string {
	return op.name
}

// Call is part of interface terex.Operator
func (op *bidiTreeOp) Call(el terex.Element, env *terex.Environment) terex.Element {
	return terex.Elem(nil)
}

// Quote is part of interface terex.Operator
func (op *bidiTreeOp) Quote(el terex.Element, env *terex.Environment) terex.Element {
	return el
}

func newBidiTreeOp(name string) *bidiTreeOp {
	return &bidiTreeOp{
		name: name,
	}
}

var _ terex.Operator = newBidiTreeOp("Hello")
