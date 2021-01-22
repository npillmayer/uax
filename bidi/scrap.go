package bidi

import (
	"fmt"

	"golang.org/x/text/unicode/bidi"
)

type charpos uint32 // position of a character within a paragraph

// --- Scraps ----------------------------------------------------------------

// Our parser and the CSG actions resemble a system developed by Prof. Donald E. Knuth
// called "WEB"/"CWEB". This systems uses context sensitive rules to style
// Pascal- and C-source code with the TeX typesetting system. The source code of
// CWEB is accessible with every TeX-installation.
//
// WEB/CWEB handles grammar tokens slightly similar to the bidi class clusters in
// our algorithm and calls these tokens `scraps'. UAX#9 does not offer a name for
// character clusters, as it handles single characters. During programming the
// first draft of the bidi algorithm I kept being reminded of Prof Knuth's `scraps'
// and finally stuck to the name.
//
// A scrap may be imagined as an interval with a bidi class value. Interval boundaries
// are positions in the input text. Scrap intervals may grow by being melded together
// with other scraps, e.g. two consecutive L-scraps (denoting writing direction
// Left-to-right) will become a single L-scrap spanning the two scraps.
//
//    [15 L 23] [23 L 148]   ⇒   [15 L 148]
//
// The resulting scrap represents a text interval with left-to-right direction.

type scrap struct {
	bidiclz  bidi.Class // bidi character class of this scrap
	l, r     charpos    // left and right bounds, r not included
	context  dirContext // directional context
	children [][]scrap  // may build up a tree of isolating run sequences
}

func (s scrap) String() string {
	if s.bidiclz == cBRACKO || s.bidiclz == cBRACKC {
		return fmt.Sprintf("[%d.%s]", s.l, classString(s.bidiclz))
	}
	if s.l == s.r-1 { // interval of length 1
		return fmt.Sprintf("[%d.%s]", s.l, classString(s.bidiclz))
	}
	if s.l == s.r { // interval of length 0
		return fmt.Sprintf("|%s|", classString(s.bidiclz))
	}
	return fmt.Sprintf("[%d-%s-%d]", s.l, classString(s.bidiclz), s.r)
}

// clear initializes a scrap to neutral values.
func (s *scrap) clear() {
	s.l, s.r = 0, 0
	s.bidiclz = cNULL
	s.context = dirContext{}
	s.children = nil
}

func (s *scrap) appendChild(ch []scrap) {
	if ch == nil {
		return
	}
	if s.children == nil {
		s.children = make([][]scrap, 0, 2)
	}
	s.children = append(s.children, ch)
}

func (s *scrap) appendAllChildrenOf(other scrap) {
	if other.children == nil || len(other.children) == 0 {
		return
	}
	for _, ch := range other.children {
		s.appendChild(ch)
	}
}

// len returns the length in bytes for a scrap.
func (s scrap) len() charpos {
	if s.bidiclz == cNULL {
		return 0
	}
	return s.r - s.l
}

func (s scrap) contains(pos charpos) bool {
	return s.l <= pos && s.r > pos
}

// collapse unifies two input scraps to a single resulting scrap with
// bidi class c.
func collapse(dest, src scrap, c bidi.Class) scrap {
	//x := dest
	dest.appendAllChildrenOf(src)
	dest.r = src.r
	dest.bidiclz = c
	//T().Debugf("%s + %s = %s", x, src, dest)
	return dest
}

// return the embedding direction for this scrap
func (s scrap) e() bidi.Class {
	return s.context.EmbeddingDir()
}

// return the opposite direction for this scrap
func (s scrap) o() bidi.Class {
	return opposite(s.context.EmbeddingDir())
}

func opposite(dir bidi.Class) bidi.Class {
	if dir == bidi.L {
		return bidi.R
	}
	if dir == bidi.R || dir == bidi.AL {
		return bidi.L
	}
	return cNI
}

// func (s scrap) LocalStrongContext(other scrap) bidi.Class {
// 	if s.context.pos >= other.l {
// 		return s.context.Context()
// 	}
// 	return NI
// }

func (s scrap) Context() bidi.Class {
	return s.context.Context()
}

func (s scrap) StrongContext() bidi.Class {
	return s.context.StrongContext()
}

func (s scrap) HasEmbeddingMatchAfter(other scrap) bool {
	return s.context.matchPos >= other.l
}

func (s scrap) HasOppositeAfter(other scrap) bool {
	pos := s.context.matchPos + charpos(s.context.odist)
	return pos >= other.l
}

// --- Strong types context --------------------------------------------------

// dirContext is a helper type to store positions of strong types within the
// input text. Various UAX#9 rules require to find preceding occurences of strong types
// (L, R, sos, AL) to determine context. In order to avoid travelling the text backwards
// we save the positions of strong types.
//
// This is quite a memory invest. We have to strike a balanced trade-off between
// speed and space efficiency. The information stored in dirContext is mainly used
// for the following rules:
//
// * Rule W2, which changes numbers to arabic numbers if there is a recent strong
//   type of AL (arabic letter)
// * Rule W7, which changes numbers to type L if there is a recent strong type of L
// * Rule N0, the handling of bracket pairings, which uses information about recent
//   strong types L or R to decide the embedding level of brackets
//
// Rule W2 could be handled by a boolean flag, but the combination of W7 and N0 is
// more tricky (see the code dealing with bracket pairs).
//
type dirContext struct {
	embeddingDir bidi.Class // embedding context
	strong       bidi.Class // current strong context
	odist        uint16     // distance between matchPos and occurrence of recent o scrap
	matchPos     charpos    // most recent position matching the embedding dir
}

// Context is either the strong context or, if that is neutral, the embedding context.
func (dc dirContext) Context() bidi.Class {
	if dc.strong == cNI {
		return dc.embeddingDir
	}
	if dc.strong == bidi.AL {
		return bidi.R
	}
	return dc.strong
}

// Has the last strong type been L or R?
func (dc dirContext) StrongContext() bidi.Class {
	if dc.strong == bidi.AL {
		return bidi.R
	}
	return dc.strong
}

// Embedding direction, as determined by the last LRI or RLI.
func (dc dirContext) EmbeddingDir() bidi.Class {
	return dc.embeddingDir
}

// Set position of recent strong type. If it matches the embedding direction
// and at > the previous matching position, the matching position is updated.
func (dc dirContext) SetStrongType(c bidi.Class, at charpos) dirContext {
	if c != bidi.L && c != bidi.R && c != bidi.AL && c != cNI {
		return dc
	}
	dc.strong = c
	if c == dc.embeddingDir && at > dc.matchPos {
		dc.matchPos = at
	} else if c == opposite(dc.embeddingDir) && at > dc.matchPos {
		d := at - dc.matchPos
		if d > 65535 {
			T().Errorf("overflow for opposite-char distance: %d", d)
			d = 65535
		}
		dc.odist = uint16(d)
	}
	T().Debugf("setting strong type %s at pos=%d, context=%v", classString(c), at, dc)
	return dc
}

// Has the currently last strong type been an AL?
func (dc dirContext) IsAL() bool {
	return dc.strong == bidi.AL
}

func (dc dirContext) SetEmbedding(dir bidi.Direction) dirContext {
	if dir == bidi.LeftToRight {
		dc.embeddingDir = bidi.L
	} else if dir == bidi.RightToLeft {
		dc.embeddingDir = bidi.R
	}
	return dc
}
