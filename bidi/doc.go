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
as operations on “look-behinds”, or as setting properties per character (Bidi class,
embedding level) or a multi-pass approach. This package employs strategies borrowed
from parsing theory to arrive at the same results as the original UAX#9 algorithm.

That said, this package will implement UAX#9 in a way that conforms to the standard
for “reasonable texts”, i.e. text produced by humans for humans. Deviation from the
standard is confined to areas of the standard that deal with rather obscure border
cases. As an example, the Bidi Annex postulates a clear maximum nesting level of
bracket pairings (63 levels) per isolating run sequence. However, this package
will ignore this boundary in a certain case when markers ending an isolating run
sequence go missing. The only clients to ever recognize this deviation are most
probably UAX#9 conformity tests.

There is one limitation, however, which ignores the standard in a relevant way:
We do not implement legacy formatting directives, which the Annex calls
“Explicit Directional Embedding and Override Formatting Characters”, i.e. the
formatting directives LRE, RLE, LRO, RLO and PDF. Unicode recommends sticking
to the more modern “Isolate Formatting Characters” LRI, RLI, FSI and PDI.
This package will deal with isolate run sequences produces by isolate formatting
characters (or external markup) only. The need to deal with legacy formatting
characters may arise in the future, but currently I do not plan to implement them.

As the algorithms in this package will not copy any input characters, it leaves
the burden to store the text to the calling client. This package will return
Bidi runs as intervals of text positions, which means clients must be able to
reproduce the text identified by text position. That's trivially true for text
stored in a bytes buffer or string, but one can imagine other situations where
this requirement involves some additional effort, like an input stream read from
a file.

Attention: Work in progress, not yet fully functional.

________________________________________________________________________________

BSD License

Copyright © 2019–2021, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. */
package bidi

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to a global core tracer
func T() tracing.Trace {
	return gtrace.CoreTracer
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
