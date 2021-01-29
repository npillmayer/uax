/*
Package grapheme implements Unicode Annex #29 grapheme breaking.

UAX#29 is the Unicode Annex for breaking text into graphemes, words
and sentences.
It defines code-point classes and sets of rules
for how to place break points and break inhibitors.
This file is about grapheme breaking.

Typical Usage with a Segmenter

Clients instantiate a grapheme object and use it as the
breaking engine for a segmenter.

  onGraphemes := grapheme.NewBreaker()
  segmenter := uax.NewSegmenter(onGraphemes)
  segmenter.Init(…)
  for segmenter.Next() {
      grphm := segmenter.Bytes()
      …
  }

Grapheme Strings

This package provides an additional convenience type `grapheme.String`.
Grapheme strings are a read-only data structure and not intended for large
texts, but rather for small to medium-sized strings. For larger texts
clients should use a segmenter.

	s := grapheme.StringFromString("世界")
	fmt.Printf("number of graphemes: %s", s.Len())                      // => 2
	fmt.Printf("number of bytes for 2nd grapheme: %d", len(s.Nth(1)))   // => 3

Attention

Before using grapheme breakers, clients will have to initialize the
classes and rules.

  SetupGraphemeClasses()

This initializes all the code-point range tables. Initialization is
not done beforehand, as it consumes quite some memory.
As grapheme breaking involves knowledge of emoji classes, a call to
SetupGraphemeClasses() will in turn call SetupEmojisClasses().

Usage of grapheme.String will take care of doing the setup behind the scenes.

Conformance

This UnicodeBreaker successfully passes all 672 tests for grapheme
breaking of UAX#29 (GraphemeBreakTest.txt).

____________________________________________________________________________

BSD License

Copyright (c) 2017–21, Norbert Pillmayer

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
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRETC, INDIRETC, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRATC, STRITC LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package grapheme

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// TC traces to the core-tracer.
func TC() tracing.Trace {
	return gtrace.CoreTracer
}
