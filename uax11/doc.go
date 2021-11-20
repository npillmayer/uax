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

License

This project is provided under the terms of the UNLICENSE or
the 3-Clause BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: 'Unlicense' OR 'BSD-3-Clause'

You may use the project under the terms of either license.

Licenses are reproduced in the license file in the root folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>


*/
package uax11

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces to uax.segment .
func tracer() tracing.Trace {
	return tracing.Select("uax.segment")
}
