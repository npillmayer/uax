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
breaking of UAX#29 (GraphemeBreakTest.txt). UPDATE: Due to a small change in
the segmenters semantics, currently 11 out of 672 tests fail. I did not have
the time to look into it.

____________________________________________________________________________

License

This project is provided under the terms of the UNLICENSE or
the 3-Clause BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: 'Unlicense' OR 'BSD-3-Clause'

You may use the project under the terms of either license.

Licenses are reproduced in the license file in the root folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package grapheme

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces to uax.segment .
func tracer() tracing.Trace {
	return tracing.Select("uax.segment")
}

// Version is the Unicode version this package conforms to.
const Version = "11.0.0"
