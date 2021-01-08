package bidi

import (
	"fmt"

	"golang.org/x/text/unicode/bidi"
)

// --- Scraps ----------------------------------------------------------------

type charpos uint32 // position of a character within a paragraph

type scrap struct {
	bidiclz bidi.Class  // bidi character class of this scrap
	l, r    charpos     // left and right bounds, r not included
	strong  strongTypes // positions of last strong bidi characters
	child   *scrap      // some scraps may have a child worth saving
}

func (s scrap) String() string {
	if s.bidiclz == BRACKO || s.bidiclz == BRACKC {
		return fmt.Sprintf("[%d.%s]", s.l, ClassString(s.bidiclz))
	}
	if s.l == s.r-1 { // interval of length 1
		return fmt.Sprintf("[%d.%s]", s.l, ClassString(s.bidiclz))
	}
	if s.l == s.r { // interval of length 0
		return fmt.Sprintf("|%s|", ClassString(s.bidiclz))
	}
	return fmt.Sprintf("[%d-%s-%d]", s.l, ClassString(s.bidiclz), s.r)
}

// clear initializes a scrap to neutral values.
func (s *scrap) clear() {
	s.l, s.r = 0, 0
	s.bidiclz = NULL
	s.strong = [4]uint16{}
	s.child = nil
}

// len returns the length in bytes for a scrap.
func (s scrap) len() charpos {
	if s.bidiclz == NULL {
		return 0
	}
	if s.bidiclz == BRACKO || s.bidiclz == BRACKC {
		return 1
	}
	return s.r - s.l
}

// --- Strong types bitfield -------------------------------------------------

const ( // positions within type strongTypes:
	lpart  uint64 = iota // position of last L
	rpart                // position of last R
	alpart               // position of last AL
	embed                // embedding direction
)

// strongTypes is a helper type to store positions of strong types within the
// input text. Various UAX#9 rules require to find preceding occurences of strong types
// (L, R, sos, AL) to determine context. In order to avoid travelling the text backwards
// we save the positions of strong types.
//
// This is quite a memory invest, but we try to manage it by storing 4 pieces of
// information in one 64 bit memory word. We hold that positions of characters within a
// paragraph of text will not overflow uint16, which is ~32.000 bytes. That should be
// enough for all but machine generated paragraphs, even when encoding non-Western languages.
// However, we make sure the scanner doesn't break in case of overflow, but rather
// will muddle along reasonably well (no panic, memory fault, etc). This is not
// a difficult task, as just taking the low bits will do just fine, except for
// handling of bracket pairs.
//
// The memory layout will be like this:
//
//    +--------------+--------------+--------------+--------------+
//    |  emb.dir.    |    AL pos.   |     R pos.   |    L pos.    |
//    +--------------+--------------+--------------+--------------+
//   64                            32                             0
//
type strongTypes [4]uint16

// Position of last L and R, respectively.
func (st strongTypes) LRPos() (int, int) {
	return int(st[lpart]), int(st[rpart])
}

// Has the last strong type been L or R?
func (st strongTypes) Context() bidi.Class {
	if st[lpart] >= st[rpart] {
		return bidi.L
	}
	return bidi.R
}

// Embedding direction, as determined by the last LRI or RLI.
func (st strongTypes) EmbeddingDir() bidi.Class {
	return bidi.Class(st[embed])
}

// Set position of L strong type.
func (st strongTypes) SetLDist(d charpos) strongTypes {
	st[lpart] = uint16(d)
	return st
}

// Set position of R strong type.
func (st strongTypes) SetRDist(d charpos) strongTypes {
	st[rpart] = uint16(d)
	return st
}

// Set position of AL.
func (st strongTypes) SetALDist(d charpos) strongTypes {
	st[alpart] = uint16(d)
	return st
}

// Has the currently last strong type been an AL?
func (st strongTypes) IsAL() bool {
	return st[alpart] > st[lpart] && st[alpart] > st[rpart]
}
