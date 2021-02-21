/*
Package uax11 provides utilities for Unicode® Standard Annex #11 “East Asian Width”.

UAX 11 Introduction

This annex presents the specifications of a normative property for Unicode characters
that is useful when interoperating with East Asian Legacy character sets.
[…] When dealing with East Asian text, there is the concept of an inherent width of a
character. This width takes on either of two values: narrow or wide.

[…]

For a traditional East Asian fixed pitch font, this width translates to a display
width of either one half or a whole unit width. A common name for this unit width
is “Em”. While an Em is customarily the height of the letter “M”, it is the same as
the unit width in East Asian fonts, because in these fonts the standard character cell
is square

[…]

Except for a few characters, which are explicitly called out as fullwidth or halfwidth
in the Unicode Standard, characters are not duplicated based on distinction in width.
Some characters, such as the ideographs, are always wide; others are always narrow;
and some can be narrow or wide, depending on the context. The Unicode character
property East_Asian_Width provides a default classification of characters, which
an implementation can use to decide at runtime whether to treat a character as narrow
or wide.

Caveats

Determining the legacy fixed-width display length is not an exact science.
Much depends on the properties of output devices, on fonts used, on a device's
interpretation of display rules, etc. Clients should treat results of UAX#11
as heuristics. Using proportional fonts is almost always a better solution.

___________________________________________________________________________

BSD License

Copyright © 2021, Norbert Pillmayer

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
package uax11

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to a global core tracer
func T() tracing.Trace {
	return gtrace.CoreTracer
}
