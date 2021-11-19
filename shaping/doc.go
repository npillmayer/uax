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

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

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
