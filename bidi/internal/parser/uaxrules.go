package parser

/*
Scanner:
 ✓  L  EN → L                 // e.g., variable names: "var1"
 ✓  WS    → NI                // whitespace to neutral
 ✓  S     → NI                //
 ✓  NI ON → NI                // except brackets

W1.
 ✓  AL  NSM NSM → AL  AL  AL  // done by scanner
 ✓  sos NSM     → sos R       // done by scanner
 -  LRI NSM     → LRI ON      // no overflow LRIs, etc.
 -  PDI NSM     → PDI ON      //

W2.
 ✓  AL … EN     → AL … AN     // done by the scanner

W3.
 ✓  Change all ALs to R.

W4.
 ✓  EN ES EN → EN EN EN
 ✓  EN CS EN → EN EN EN
 ✓  AN CS AN → AN AN AN

W5.
 ✓  ET ET EN → EN EN EN
 ✓  EN ET ET → EN EN EN
 -  AN ET EN → AN EN EN       // nothing to do

W6.
 ✓  AN ET    → AN ON
 ✓  L  ES EN → L  ON EN
 ✓  EN CS AN → EN ON AN
 ✓  ET AN    → ON AN

W7.
 ✓  L  NI EN → L  NI  L      // prepared by scanner as   L NI LEN  → L   (= W7+N1)

---

N1.
 ✓   L  NI   L  →   L  L   L
 ✓   R  NI   R  →   R  R   R
 ✓   R  NI  AN  →   R  R  AN
     R  NI  EN  →   R  R  EN
    AN  NI   R  →  AN  R   R    // rewriter
    AN  NI  AN  →  AN  R  AN    // rewriter
    AN  NI  EN  →  AN  R  EN    // rewriter
    EN  NI   R  →  EN  R   R
    EN  NI  AN  →  EN  R  AN
    EN  NI  EN  →  EN  R  EN


*/

// --- EBNF Grammar ----------------------------------------------------------

// This is code, please to not edit!

//go:generate grammy -grammar BiDi -hook bidiToken -package github.com/npillmayer/uax/bidi
// Run         = "(" name ")" .

func bidiToken(t string) (string, int) {
	// TOOD
	return t, -100
}

// Production  = name "=" [ Expression ] "." .
// Expression  = Alternative { "|" Alternative } .
// Alternative = Term { Term } .
// Term        = name | token [ "…" token ] | Group | Option | Repetition .
// Group       = "(" Expression ")" .
// Option      = "[" Expression "]" .
// Repetition  = "{" Expression "}" .
