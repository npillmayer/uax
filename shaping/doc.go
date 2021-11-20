/*
Package shaping provides tables corresponding to Unicode® Character Data tables relevant
for text shaping.

Most of the tables have been created by a generator CLI (source located in
github.com/npillmayer/uax/internal/tablegen).
Parameters for calling tablegen are as follows:

▪︎ arabictables.go:

	tablegen -f 3 -p shaping -o arabictables.go -x ARAB
	         -u https://www.unicode.org/Public/13.0.0/ucd/ArabicShaping.txt

▪︎ uipctables.go:

	tablegen -f 2 -p shaping -o uipctables.go -x UIPC
	         -u https://www.unicode.org/Public/13.0.0/ucd/IndicPositionalCategory.txt

▪︎ uisctables.go:

	tablegen -f 2 -p shaping -o uisctables.go -x UISC
	         -u https://www.unicode.org/Public/13.0.0/ucd/IndicSyllabicCategory.txt

___________________________________________________________________________

License

This project is provided under the terms of the UNLICENSE or
the 3-Clause BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: 'Unlicense' OR 'BSD-3-Clause'

You may use the project under the terms of either license.

Licenses are reproduced in the license file in the root folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>


*/
package shaping

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces to a global core tracer
func tracer() tracing.Trace {
	return gtrace.CoreTracer
}
