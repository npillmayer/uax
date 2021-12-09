/*
Package bidi will implement a variant of the Unicode UAX#9 Bidirectional Algorithm.
It is not fully standards-conforming, but good enough for practical purposes.

Unicode Annex UAX#9 presents an algorithm to identify directional runs within
texts. The algorithm deals with characters and character runs, which UAX#9
maps to Bidi character classes. Bidi classes are then grouped according to
certain rules to determine writing directions. The algorithm is not perfect and
there are some cases where manual overriding will be necessary to produce correct
output, but it is good enough for many real-life cases.

This package will interpret some of the Bidi algorithm's rules a bit differently
than a strict adhering to the standard would require, the reason being that we
postulate some general requirements which make it hard to conform to the standard
100%. The main general requirement is a restriction of the mode of access for the
input text: We operate on an `io.Reader` and do not buffer the characters read from
it. As a consequence, we will never travel backwards over characters and will never
read a character twice. However, some parts of the UAX#9 algorithm are presented
as operations on “look-behinds,” or as setting properties per character (Bidi class,
embedding level) or as a multi-pass approach. This package employs strategies borrowed
from parsing theory to arrive at the same results as the original UAX#9 algorithm.

Deviations from the Standard

This package implements UAX#9 in a way that is standards-conforming for
“reasonable texts”, i.e. text produced by humans for humans. Deviations from the
standard are confined to two areas of the standard: error handling and legacy
formatting directives.

As an example for error handling, the Bidi Annex postulates a clear maximum nesting
level of bracket pairings (63 levels) per isolating run sequence. However, this package
will ignore this boundary in a certain case when markers ending an isolating run
sequence go missing. The only clients to ever recognize this deviation are most
probably UAX#9 conformity tests.

We do not implement legacy formatting directives, which the Annex calls
“Explicit Directional Embedding and Override Formatting Characters”, i.e. the
formatting directives LRE, RLE, LRO, RLO and PDF. Unicode recommends sticking
to the more modern “Isolate Formatting Characters” LRI, RLI, FSI and PDI.
This package will deal with isolate run sequences produced by isolate formatting
characters (or external markup) only. The need to deal with legacy formatting
characters may arise in the future, but currently I do not plan to implement them.

API

As the algorithms in this package will not copy any input characters, it leaves
the burden to store the text to the calling client. This package will return
Bidi runs as intervals of text positions, which means clients must be able to
reproduce the text identified by text position. That's trivially true for text
stored in a bytes buffer or string, but one can imagine other situations where
this requirement involves some additional effort, like an input stream read from
a file.

Attention: Work in progress, not yet fully functional.

________________________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package bidi

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces to uax.bidi
func tracer() tracing.Trace {
	return tracing.Select("uax.bidi")
}

// UnicodeVersion is the UAX#9 version this implementation follows.
const UnicodeVersion = "13.0.0"

// A Direction indicates the overall flow of text.
type Direction int

const (
	// LeftToRight indicates a requirement to order the characters of a script
	// from left to right.
	LeftToRight Direction = iota
	// RightToLeft indicates a requirement to order the characters of a script
	// from right to left.
	RightToLeft
)

// String returns either "L2R" or "R2L".
func (dir Direction) String() string {
	switch dir {
	case LeftToRight:
		return "L2R"
	case RightToLeft:
		return "R2L"
	}
	return "unknown direction"
}
