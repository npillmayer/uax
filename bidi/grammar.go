package bidi

import (
	"errors"
	"fmt"
	"sync"

	"github.com/npillmayer/gorgo/lr"
	"github.com/npillmayer/gorgo/lr/earley"
	"github.com/npillmayer/gorgo/lr/sppf"
	"github.com/npillmayer/gorgo/terex"
	"github.com/npillmayer/gorgo/terex/termr"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/text/unicode/bidi"
)

// --- Initialization --------------------------------------------------------

var globalBidiGrammar *lr.LRAnalysis

var initParser sync.Once

func getParser() *earley.Parser {
	initParser.Do(func() {
		globalBidiGrammar = NewBidiGrammar()
		globalBidiGrammar.Grammar().Dump()
	})
	parser := earley.NewParser(globalBidiGrammar, earley.GenerateTree(true), earley.StoreTokens(true))
	if parser == nil {
		panic("Could not created Bidi grammar parser")
	}
	initRewriters()
	return parser
}

/*
	b.LHS("Run").T(bc(bidi.LRI)).N("NISeq").N("L").N("NISeq").T(bc(bidi.PDI)).End()
	//b.LHS("Run").T(bc(bidi.LRI)).N("L").T(bc(bidi.PDI)).End()
	//b.LHS("Run").N("LRI").N("L").N("PDI").End()
	//b.LHS("Run").T(bc(bidi.RLI)).N("R").T(bc(bidi.PDI)).End()
	b.LHS("L").N("LParenRun").End()
	b.LHS("LParenRun").T(bc(LPAREN)).N("NISeq").N("L").N("NISeq").T(bc(RPAREN)).End()
	// b.LHS("R").N("RParenRun").End()
	// b.LHS("RParenRun").T(bc(LPAREN)).N("R").T(bc(RPAREN)).End()
	b.LHS("NI").T(bc(bidi.B)).End()
	b.LHS("NI").T(bc(bidi.S)).End()
	b.LHS("NI").T(bc(bidi.WS)).End()
	b.LHS("NI").T(bc(bidi.ON)).End()
	b.LHS("NI").N("NI").N("NI").End()
	b.LHS("L").T(bc(bidi.L)).End()
	b.LHS("R").T(bc(bidi.R)).End()
	// b.LHS("AL").T(bc(bidi.AL)).End()
	// b.LHS("EN").T(bc(bidi.EN)).End()
	// b.LHS("LEN").T(bc(LEN)).End()
	// b.LHS("AN").T(bc(bidi.AN)).End()
	// b.LHS("R").N("AL").End()                            // W3
	// b.LHS("EN").N("EN").T(bc(bidi.ES)).N("EN").End()    // W4
	// b.LHS("EN").N("EN").T(bc(bidi.CS)).N("EN").End()    // W4
	// b.LHS("AN").N("AN").T(bc(bidi.CS)).N("AN").End()    // W4
	// b.LHS("LEN").N("LEN").T(bc(bidi.ES)).N("LEN").End() // W4
	// b.LHS("LEN").N("LEN").T(bc(bidi.CS)).N("LEN").End() // W4
	// b.LHS("EN").N("EN").T(bc(bidi.ET)).End()            // W5
	// b.LHS("EN").T(bc(bidi.ET)).N("EN").End()            // W5
	// b.LHS("LEN").N("LEN").T(bc(bidi.ET)).End()          // W5
	// b.LHS("LEN").T(bc(bidi.ET)).N("LEN").End()          // W5
	b.LHS("NI").T(bc(bidi.CS)).End() // W6
	b.LHS("NI").T(bc(bidi.ET)).End() // W6
	b.LHS("NI").T(bc(bidi.ES)).End() // W6
	b.LHS("L").N("LEN").End()                // W7
	b.LHS("L").N("L").N("NI").N("L").End()   // N1
	b.LHS("R").N("R").N("NI").N("R").End()   // N1
	b.LHS("L").N("L").N("NI").N("LEN").End() //
	b.LHS("L").N("L").N("L").End()           //
	b.LHS("NISeq").N("NI").End()             //
	b.LHS("NISeq").Epsilon()                 //

	// LSpan is a run :LRI … :PDI, which may include mixed Ls and Rs
	b.LHS("LSpan").N("LSOS").N("EOS").End()      // LSpan may be pure L
	b.LHS("LSpan").N("LSpanFrag").N("EOS").End() // LSpan must include EOS(=PDI)
	// LSpanFrag is a fragment :LRI …, which included at least one R
	b.LHS("LSpanFrag").N("LSpanFrag").N("NISeq").N("L").End()
	b.LHS("LSpanFrag").N("LSpanFrag").N("NISeq").N("R").End()
	b.LHS("LSpanFrag").N("LSOS").N("R").N("NISeq").N("L").End() // bridge R
	b.LHS("LSpanFrag").N("LSOS").N("R").End()
	// LSOS is a fragment :LRI …, which does not include any Rs
	b.LHS("LSOS").N("LSOS").N("NI").End() // LSOS gobbles up Ls and NIs (no Rs)
	b.LHS("LSOS").N("LSOS").N("L").End()
	b.LHS("LSOS").T(bc(bidi.LRI)).N("L").End()  // LSOS starts at LRI
	b.LHS("LSOS").T(bc(bidi.LRI)).N("NI").End() // TODO: EN, etc.
	// EOS is a closing fragment [NI] :PD
	b.LHS("EOS").N("NI").T(bc(bidi.PDI)).End() // EOS collects trailing NIs
	b.LHS("EOS").T(bc(bidi.PDI)).End()
	//
	b.LHS("L").N("LBrackFrag").N("BrackC").End() // an L run between matching brackets
	b.LHS("L").N("LMixedFrag").N("BrackC").End()
	//
	b.LHS("LMixedFrag").N("LBrackFrag").N("NI").End()
	b.LHS("LMixedFrag").N("LBrackFrag").N("L").End()
	b.LHS("LMixedFrag").N("LBrackFrag").N("R").End()
	//
	b.LHS("LBrackFrag").N("LBrackFrag").N("NI").End() // LBrackFrag gobbles up Ls and NIs (no Rs)
	b.LHS("LBrackFrag").N("LBrackFrag").N("L").End()
	b.LHS("LBrackFrag").T(bc(LBRACKO)).N("L").End()  // LBrackFrag starts at :LBRACKO
	b.LHS("LBrackFrag").T(bc(LBRACKO)).N("NI").End() // TODO: EN, etc.
	// BrackC is a closing fragment [NI] :BRACKC
	b.LHS("BrackC").N("NI").T(bc(BRACKC)).End() // BRACKC collects trailing NIs
	b.LHS("BrackC").T(bc(BRACKC)).End()
*/

// NewBidiGrammar creates a new grammar for parsing Bidi runs in a paragraph. It is usually
// not called by clients directly, but rather used transparently with a call to Parse.
// It is included in the API for advanced usage, like extending or modifying the grammar.
func NewBidiGrammar() *lr.LRAnalysis {
	b := lr.NewGrammarBuilder("UAX#9")

	//b.LHS("S").T(bc(bidi.LRI)).N("R").T(bc(bidi.PDI)).End() // for testing
	//
	b.LHS("Para").N("LRun").End() // L to R paragraph
	// b.LHS("NI").N("LRun").End()            // explicit runs are neutral to surrounding text
	b.LHS("LRun").N("LSOS").N("EOS").End() //
	// //
	b.LHS("LSOS").T(bc(bidi.LRI)).End()      // LSOS starts at LRI
	b.LHS("LSOS").N("LSOS").N("NISeq").End() // LSOS gobbles up NIs
	b.LHS("LSOS").N("LSOS").N("L").End()     // LSOS gobbles up Ls
	b.LHS("LSOS").N("LSOS").N("R").End()     //
	// //
	b.LHS("EOS").T(bc(bidi.PDI)).End()
	b.LHS("EOS").N("NISeq").N("EOS").End() // EOS collects trailing NIs
	//
	// //b.LHS("R").N("R").N("NISeq").N("EN").N("NISeq").N("R").End()
	//
	// //b.LHS("L").N("L").N("L").End() // really needed and possible
	//
	// N0. Process bracket pairs
	// try to keep bracket information
	b.LHS("L").N("LBrackRun").End()
	b.LHS("LBrackRun").N("LBrackO").N("L").N("LBrackC").End()
	b.LHS("LBrackO").T(bc(LBRACKO)).End()
	b.LHS("LBrackO").N("LBrackO").N("NI").End()
	b.LHS("LBrackC").N("NI").N("LBrackC").End()
	b.LHS("LBrackC").T(bc(LBRACKC)).End()
	//
	// 3.3.4 Resolving Weak Types
	b.LHS("R").T(bc(bidi.AL)).End() // W3: AL → R
	//b.LHS("EN").N("EN").T(bc(bidi.ES)).T(bc(bidi.EN)).End() // W4: EN ES EN → EN EN EN
	b.LHS("LEN").N("LEN").T(bc(bidi.ES)).N("LEN").End() // W4  "
	//b.LHS("EN").N("EN").T(bc(bidi.CS)).T(bc(bidi.EN)).End() // W4: EN CS EN → EN EN EN
	b.LHS("LEN").N("LEN").T(bc(bidi.CS)).N("LEN").End() // W4  "
	//b.LHS("AN").N("AN").T(bc(bidi.CS)).T(bc(bidi.AN)).End() // W4: AN CS AN → AN AN AN
	//b.LHS("EN").T(bc(bidi.EN)).T(bc(bidi.ET)).End()         // W5: EN ET ET → EN EN EN
	//b.LHS("EN").N("EN").T(bc(bidi.ET)).End()                // W5  "
	b.LHS("LEN").T(bc(LEN)).T(bc(bidi.ET)).End() // W5  "
	b.LHS("LEN").N("LEN").T(bc(bidi.ET)).End()   // W5  "
	//b.LHS("EN").T(bc(bidi.ET)).T(bc(bidi.EN)).End()         // W5: ET ET EN → EN EN EN
	//b.LHS("EN").T(bc(bidi.ET)).N("EN").End()                // W5  "
	// //b.LHS("LEN").T(bc(bidi.ET)).T(bc(LEN)).End() // W5  "
	b.LHS("LEN").T(bc(bidi.ET)).N("LEN").End() // W5  "
	b.LHS("ON").T(bc(bidi.CS)).End()           // W6  convert unused number tokens
	b.LHS("ON").T(bc(bidi.ET)).End()           // W6  convert unused number tokens
	b.LHS("ON").T(bc(bidi.ES)).End()           // W6  convert unused number tokens
	//
	// // b.LHS("L").N("L").N("NISeq").T(bc(LEN)).End() // W7+N1: L NI EN → L NI L → L
	//b.LHS("L").N("L").T(bc(NI)).T(bc(LEN)).End()            // W7+N1: "
	b.LHS("L").N("L").N("NISeq").N("LEN").End() // W7+N1: "
	b.LHS("EN").T(bc(bidi.EN)).End()            //
	b.LHS("LEN").T(bc(LEN)).End()               //
	//
	// 3.3.5 Resolving Neutral and Isolate Formatting Types
	//b.LHS("L").T(bc(bidi.L)).N("NISeq").T(bc(bidi.L)).End() // N1
	b.LHS("L").N("L").N("NISeq").N("L").End() // N1
	//b.LHS("R").T(bc(bidi.R)).N("NISeq").T(bc(bidi.R)).End() // N1
	b.LHS("R").N("R").N("NISeq").N("R").End()  // N1
	b.LHS("R").N("R").N("NISeq").N("AN").End() // N1
	//b.LHS("R").T(bc(bidi.R)).N("NISeq").N("AN").End()       // N1
	//
	b.LHS("R").N("AN").N("NISeq").N("R").End()  // N1
	b.LHS("R").N("AN").N("NISeq").N("AN").End() // N1
	b.LHS("R").N("AN").N("NISeq").N("EN").End() // N1
	//
	b.LHS("R").N("EN").N("NISeq").N("R").End()  // N1
	b.LHS("R").N("EN").N("NISeq").N("AN").End() // N1
	b.LHS("R").N("EN").N("NISeq").N("EN").End() // N1
	//
	// Squash sequence of NIs to single NI. NIs may have been produced from
	// other punctuation or from runs.
	b.LHS("NISeq").N("NISeq").N("NI").End() //
	b.LHS("NISeq").N("NI").End()            //
	//b.LHS("NISeq").Epsilon()                // try to avoid eps rules
	//
	// Base Types
	b.LHS("L").T(bc(bidi.L)).End()
	b.LHS("R").T(bc(bidi.R)).End()
	//
	b.LHS("NI").N("NI").T(bc(bidi.ON)).End() // frequent case
	b.LHS("NI").T(bc(NI)).End()
	b.LHS("NI").T(bc(bidi.ON)).End()
	b.LHS("NI").T(bc(bidi.ET)).End()
	b.LHS("NI").T(bc(bidi.ES)).End()
	b.LHS("NI").T(bc(bidi.CS)).End()
	//
	g, err := b.Grammar()
	if err != nil {
		panic(err)
	}
	return lr.Analysis(g)
}

// Parse parses a paragraph of text, given by a scanner, and parses it according to the
// Unicode UAX#9 Bidi algorithm.
func Parse(scanner *Scanner) (bool, *sppf.Forest, error) {
	if scanner == nil {
		return false, nil, errors.New("Expected parameter scanner to be non-nil")
	}
	parser := getParser()
	var parsetree *sppf.Forest
	accept, err := parser.Parse(scanner, nil)
	if accept {
		parsetree = parser.ParseForest()
	}
	return accept, parsetree, err
}

func bc(tokval bidi.Class) (string, int) {
	return ":" + ClassString(tokval), int(tokval)
}

// --- AST creation ----------------------------------------------------------

// AST creates an abstract syntax tree from a parse tree/forest.
//
// Returns a homogenous AST, a TeREx-environment and an error status.
func AST(parsetree *sppf.Forest, tokRetr termr.TokenRetriever) (*terex.GCons,
	*terex.Environment, error) {
	if globalBidiGrammar == nil {
		panic("AST creation called for BiDi, but grammar is not initialized.\nDid you skip to call the parser first?")
	}
	ab := newASTBuilder(globalBidiGrammar.Grammar())
	env := ab.AST(parsetree, tokRetr)
	if env == nil {
		T().Errorf("Cannot create AST from parsetree")
		return nil, nil, fmt.Errorf("Error while creating AST")
	}
	ast := env.AST
	T().Infof("AST: %s", env.AST.ListString())
	return ast, env, nil
}

// NewASTBuilder returns a new AST builder for the BiDi pseudo language
func newASTBuilder(grammar *lr.Grammar) *termr.ASTBuilder {
	ab := termr.NewASTBuilder(grammar)
	ab.AddTermR(lOp)
	ab.AddTermR(rOp)
	ab.AddTermR(paraOp)
	ab.AddTermR(lenOp)
	ab.AddTermR(lbrackOp)
	ab.AddTermR(anOp)
	return ab
}

// Term Rewriters

// --- Parse tree to AST rewriters -------------------------------------------

type bidiTermR struct {
	name    string
	rewrite func(*terex.GCons, *terex.Environment) terex.Element
	call    func(terex.Element, *terex.Environment) terex.Element
}

var _ terex.Operator = &bidiTermR{}
var _ termr.TermR = &bidiTermR{}

func makeASTTermR(name string) *bidiTermR {
	termr := &bidiTermR{
		name: name,
	}
	return termr
}

func (bidiw *bidiTermR) String() string {
	return bidiw.name
}

func (bidiw *bidiTermR) Operator() terex.Operator {
	return bidiw
}

func (bidiw *bidiTermR) Rewrite(l *terex.GCons, env *terex.Environment) terex.Element {
	T().Debugf("%s:bidiw.Rewrite[%s] called", bidiw.String(), l.ListString())
	e := bidiw.rewrite(l, env)
	return e
}

func (bidiw *bidiTermR) Descend(sppf.RuleCtxt) bool {
	return true
}

func (bidiw *bidiTermR) Call(e terex.Element, env *terex.Environment) terex.Element {
	panic("pmmp term rewriter not intended to be 'call'ed")
	//return callFromEnvironment(bidiw.opname, e, env)
}

// --- Init global rewriters -------------------------------------------------

var lOp *bidiTermR      // for L -> … productions
var rOp *bidiTermR      // for R -> … productions
var paraOp *bidiTermR   // for Para -> … productions
var lenOp *bidiTermR    // for LEN -> … productions
var lbrackOp *bidiTermR // for LBrackRun -> … productions
var anOp *bidiTermR     // for AN -> … productions

func initRewriters() {
	paraOp = makeASTTermR("Para")
	paraOp.rewrite = func(l *terex.GCons, env *terex.Environment) terex.Element {
		T().Infof("Para tree = ")
		terex.Elem(l).Dump(tracing.LevelInfo)
		return terex.Elem(l)
	}
	lOp = makeASTTermR("L")
	lOp.rewrite = func(l *terex.GCons, env *terex.Environment) terex.Element {
		T().Infof("L tree = ")
		terex.Elem(l).Dump(tracing.LevelInfo)
		return terex.Elem(l)
	}
	rOp = makeASTTermR("R")
	rOp.rewrite = func(l *terex.GCons, env *terex.Environment) terex.Element {
		T().Infof("R tree = ")
		terex.Elem(l).Dump(tracing.LevelInfo)
		return terex.Elem(l)
	}
	lbrackOp = makeASTTermR("LBrackRun")
	lbrackOp.rewrite = func(l *terex.GCons, env *terex.Environment) terex.Element {
		T().Infof("LBrackRun tree = ")
		terex.Elem(l).Dump(tracing.LevelInfo)
		return terex.Elem(l)
	}
	lenOp = makeASTTermR("LEN")
	lenOp.rewrite = func(l *terex.GCons, env *terex.Environment) terex.Element {
		T().Infof("LEN tree = ")
		terex.Elem(l).Dump(tracing.LevelInfo)
		return terex.Elem(l)
	}
	anOp = makeASTTermR("AN")
	anOp.rewrite = func(l *terex.GCons, env *terex.Environment) terex.Element {
		// AN  NI   R  →  AN  R   R
		// AN  NI  AN  →  AN  R  AN
		// AN  NI  EN  →  AN  R  EN
		T().Infof("AN tree = ")
		terex.Elem(l).Dump(tracing.LevelInfo)
		return terex.Elem(l)
	}
}

// ---------------------------------------------------------------------------

func earleyTokenRetriever(parser *earley.Parser) termr.TokenRetriever {
	return func(pos uint64) interface{} {
		if pos <= 1 {
			return ClassString(bidi.LRI)
		}
		return parser.TokenAt(pos)
	}
}
