/*
Package uax is about Unicode Annexes and their algorithms.

Description

From the Unicode Consortium:

A Unicode Standard Annex (UAX) forms an integral part of the Unicode
Standard, but is published online as  a  separate  document.
The Unicode Standard may require conformance to normative content
in a Unicode Standard Annex, if so specified in  the  Conformance
chapter of that version of the Unicode Standard. The version
number of a UAX document corresponds to the version of  the  Unicode
Standard of which it forms a part.

[...]

A string of Unicode‐encoded text often needs to be broken up into
text elements programmatically. Common examples of text  elements
include  what  users  think  of as characters, words, lines (more
precisely, where line breaks are  allowed),  and  sentences.  The
precise  determination of text elements may vary according to orthographic
conventions for a given script or language.  The  goal
of matching user perceptions cannot always be met exactly because
the text alone does not always contain enough information to
unambiguously  decide  boundaries.  For example, the period (U+002E
FULL STOP) is used  ambiguously,  sometimes  for  end‐of‐sentence
purposes, sometimes for abbreviations, and sometimes for numbers.
In most cases, however, programmatic text  boundaries  can  match
user  perceptions quite closely, although sometimes the best that
can be done is not to surprise the user.

[...]

There are many different ways to divide text elements corresponding
to user‐perceived characters, words, and sentences,  and  the
Unicode  Standard does not restrict the ways in which implementations
can produce these divisions.

This specification defines default mechanisms; more sophisticated
implementations can and should tailor them for particular locales
or  environments.  For example, reliable detection of word boundaries
in languages such as Thai, Lao, Chinese,  or  Japanese
requires the use of dictionary lookup, analogous to English hyphenation.

BSD License

Copyright (c) 2017–20, Norbert Pillmayer

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

Contents

Implementations of specific UAX algorithms is done in the various
sub-packages of uax. The driver type sits in sub-package segment and will
use breaking algorithms from the other sub-packages.

Base package uax provides some of the necessary
means to implement UAX breaking algorithms. Please note that it is
in now way mandatory to use the supporting types and functions of this
package. Implementors of additional breaking algorithms are free to
ignore some or all of the helpers and instead implement their breaking
algorithms from scratch.

Every implementation of UAX breaking algorithms has to handle the trade-off
between efficiency and understandability. Algorithms as described in the
Unicodes Annex documents are no easy read when considering all the details
and edge cases. Getting it 100% right therefore sometimes may be tricky.
Implementations in the sub-packages of uax try to strike a balance between
efficiency and readability. The helper classes of uax allow implementors to
transform UAX-rules into fairly readable small functions. From a maintenance
point-of-view this is preferrable to huge and complex cascades of if-statements,
which may sometimes provide better performance, but
are hard to understand. All the breaking algorithms within sub-packages of uax
therefore utilize the helper types from package uax.

We perform segmenting Unicode text based on rules, which are short
regular expressions, i.e. finite state automata. This corresponds well with
the formal UAX description of rules (expect for the bidi rules).
Every step within a
rule is performed by executing a function. This function recognizes a single
code-point class and returns another function. The returned function
represents the expectation for the next code-point(-class).
These kind of matching by function is continued until a rule is accepted
or aborted.

An example for a UAX rule is rule WB13b "Do not break from extenders"
from UAX#29:

   ExtendNumLet x (ALetter | Hebrew_Letter| Numeric | Katakana)

The 'x' denotes a suppressed break. All the identifiers are UAX#29-specific
classes for code-points. Matching them will call two functions in sequence:

      rule_WB13b( … )   // match ExtendNumLet
   -> finish_WB13b( … ) // match any of ALetter … Katakana

The final return value will either signal an accept or abort.

The uax helper type to perform this kind of matching is called Recognizer.
A set of Recognizers comprises an NFA and will match break opportunities
for a UAX rule-set. Recognizers receive rune events and therefore implement
interface RuneSubscriber.

Rune Events

Walking the runes (= code-points) of a Unicode text and firing rules to match
segments will produce a high fluctuation of short-lived Recognizers.
Every Recognizer will have to react to the next rune read. Package uax
provides a publish-subscribe mechanism for signalling new runes to all active
Recognizers.

The default rune-publisher will distribute rune events to rune-subscribers
and collect return values. Subscribers are required to return active matches
and possible break-opportunities (or suppression thereof).
After all subscribers are done consuming the rune, the publisher harvests
subscribers which have ended their life-cycle (i.e., either accepted or
aborted). Dead subscribers are flagging this with Done()==true and get
unsubscribed.

UnicodePublishers are used by the types implementing UAX breaking logic.
There's interface UnicodeBreaker, representing breaking algorithms.
The segment-driver needs one or more UnicodeBreakers to perform breaking logic.

Penalties

Algorithms in this package will signal break opportunities for Unicode text.
However, breaks are not signalled with true/false, but rather with a
weighted "penalty". Every break is connoted with an integer value,
representing the desirability of the break. Negative values denote a
negative penalty, i.e. a merit. High enough penalties signal the complete
suppression of a break opportunity, causing the segmenter to not report
this break.
The UnicodeBreakers in this package (including sub-packages)
will apply the following logic:

(1) Mandatory breaks will have a penalty/merit of -1000 (uax.InfinitePenalty)

(2) Inhibited breaks will have penalty >= 1000 (uax.InfiniteMerits)

(3) Neutral breaks will have a penalty of 0.
*/
package uax

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// CT traces to the core-tracer.
func CT() tracing.Trace {
	return gtrace.CoreTracer
}

// We define constants for flagging break points as infinitely bad and
// infinitely good, respectively.
const (
	InfinitePenalty = 1000
	InfiniteMerits  = -1000
)
