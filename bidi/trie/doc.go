/*
Package trie implements a trie data-structure similar to the one described by
Donald E Knuth in “Programming Perls”. (Communications of the ACM,
Vol. 29, No. 6, June 1986,
https://cecs.wright.edu/people/faculty/pmateti/Courses/7140/PDF/cwp-knuth-cacm-1986.pdf).

The trie is suitable for write-once-read-many-times situations. The idea is to
spend some effort to create a compact but efficient dictionary for categorical data.


License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package trie

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'uax.bidi'.
func tracer() tracing.Trace {
	return tracing.Select("uax.bid")
}
