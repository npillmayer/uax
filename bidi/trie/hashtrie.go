package trie

import "math"

type bidiclass int8
type pointer int16

// const size = 9973  // largest prime < 10000
// const catcnt = 46      // maximum number of bidi character classes
// const headercode = 47  // special code for anchor of family
const empty = 0        // denotes empty slots of the trie
const tolerance = 1000 // maximum number of trys to relocated a family

type ShortHashTrie struct {
	size       int
	catcnt     int     //bidiclass
	span       pointer // space in table without leading and trailing #catnct slots
	headercode bidiclass
	link       []pointer
	sibling    []pointer
	ch         []bidiclass
	alpha      int //pointer
}

func NewShortHashTrie(size int16, catcnt int8) *ShortHashTrie {
	trie := &ShortHashTrie{
		size:   int(size),
		catcnt: int(catcnt),
	}
	trie.headercode = bidiclass(trie.catcnt) + 1
	trie.alpha = int(math.Round(0.61803 * float64(trie.size)))
	trie.span = pointer(trie.size - 2*trie.catcnt)
	trie.setInitialValues()
	return trie
}

func (trie *ShortHashTrie) setInitialValues() {
	trie.link = make([]pointer, trie.size)
	trie.sibling = make([]pointer, trie.size)
	trie.ch = make([]bidiclass, trie.size)
	for i := 1; i <= int(trie.catcnt); i++ {
		trie.ch[i] = bidiclass(i)
		trie.sibling[i] = pointer(i) - 1
	}
	trie.ch[0] = trie.headercode
	trie.sibling[0] = pointer(trie.catcnt)
}

// findPositionForWord is Knuth's find_buffer
func (trie *ShortHashTrie) findPositionForWord(buf []byte) pointer {
	n := 0
	p := pointer(buf[0]) // current word position
	for n+1 < len(buf) {
		n++
		c := bidiclass(buf[n])
		p = trie.advanceToChild(p, c, n)
		T().Debugf("advanced to p=%d with c=%d", p, c)
	}
	return p
}

func (trie *ShortHashTrie) advanceToChild(p pointer, c bidiclass, n int) pointer {
	if trie.link[p] == 0 {
		T().Debugf("link[%d] is unassigned, inserting first child=%d", p, c)
		return trie.insertFirstbornChildAndProgress(p, c, n)
	}
	T().Debugf("position link[%d] is occupied →%d", p, trie.link[p])
	q := trie.link[p] + pointer(c)
	if trie.ch[q] != c {
		if trie.ch[q] != empty {
			p, q = trie.moveFamily(p, c, n)
		}
		q = trie.insertChildIntoFamily(p, q, c)
	}
	return q
}

func (trie *ShortHashTrie) insertFirstbornChildAndProgress(p pointer, c bidiclass, n int) pointer {
	var h pointer                             // trial header location
	var x = pointer(n*trie.alpha) % trie.span // nominal position of header #n
	//var lasth int // stopper for location search
	// Get set for computing header locations
	// if x < pointer(trie.size-2*int(trie.catcnt)-trie.alpha) {
	// 	x += pointer(trie.alpha)
	// } else {
	// 	x = x + pointer(trie.alpha-trie.size+2*int(trie.catcnt))
	// }
	h = x + pointer(trie.catcnt) + 1 // now catcnt < h ≤ (trie.size+catcnt)
	if h <= pointer(trie.catcnt) || int(h) > trie.size+trie.catcnt {
		panic("invariant not held")
	}
	// we won't use a stopper
	// if int(h) < trie.size-trie.catcnt-tolerance {
	// 	lasth = int(h) + tolerance
	// } else {
	// 	lasth = int(h) + tolerance - trie.size + 2*trie.catcnt
	// }
	//
	trys := 0
	for ; trys < tolerance; trys++ {
		// Compute the next trial header location or abort find
		if h == pointer(trie.size-trie.catcnt) {
			h = pointer(trie.catcnt + 1)
		} else {
			h++
		}
		//
		if trie.ch[h] == empty && trie.ch[h+pointer(c)] == empty {
			T().Debugf("found an empty child slot=%d→%d", h, h+(pointer(c)))
			break
		}
	}
	//if int(h) == lasth {
	if trys == tolerance {
		T().Errorf("abort find")
		panic("abort find")
	}
	trie.link[p], trie.link[h] = h, p
	p = h + pointer(c)
	trie.ch[h], trie.ch[p] = trie.headercode, c
	trie.sibling[h], trie.sibling[p] = p, h
	trie.link[p] = 0
	return p
}

// q = link[p] + c
func (trie *ShortHashTrie) insertChildIntoFamily(p, q pointer, c bidiclass) pointer {
	h := trie.link[p]
	for trie.sibling[h] > q {
		h = trie.sibling[h]
	}
	trie.sibling[q], trie.sibling[h] = trie.sibling[h], q
	trie.ch[q] = c
	trie.link[q] = 0
	T().Debugf("Inserted %d at q=%d with header=%d", c, q, trie.link[p])
	return q
}

func (trie *ShortHashTrie) moveFamily(p pointer, c bidiclass, n int) (pointer, pointer) {
	T().Debugf("have to move family for c=%d", c)
	//
	var h pointer                             // trial header location
	var x = pointer(n*trie.alpha) % trie.span // nominal position of header #n
	//var lasth int // stopper for location search
	// Get set for computing header locations
	// if x < pointer(trie.size-2*int(trie.catcnt)-trie.alpha) {
	// 	x += pointer(trie.alpha)
	// } else {
	// 	x = x + pointer(trie.alpha-trie.size+2*int(trie.catcnt))
	// }
	h = x + pointer(trie.catcnt) + 1 // now catcnt < h ≤ (trie.size+catcnt)
	if h <= pointer(trie.catcnt) || int(h) > trie.size+trie.catcnt {
		panic("invariant not held")
	}
	//
	// var h, x pointer
	// var lasth int
	// if int(x) < trie.size-2*trie.catcnt-trie.alpha {
	// 	x += pointer(trie.alpha)
	// } else {
	// 	x = x + pointer(trie.alpha-trie.size+2*trie.catcnt)
	// }
	h = x + pointer(trie.catcnt) + 1
	// if int(h) < trie.size-trie.catcnt-tolerance {
	// 	lasth = int(h) + tolerance
	// } else {
	// 	lasth = int(h) + tolerance - trie.size + 2*trie.catcnt
	// }
	//
	q := h + pointer(c)
	r := trie.link[p]
	delta := h - r
	trys := 0
	for ; trys < tolerance; trys++ {
		if h < pointer(trie.size-trie.catcnt) {
			h++
		} else {
			h = pointer(trie.catcnt) + 1
		}
		//
		if trie.ch[h+pointer(c)] != empty {
			continue
		}
		r = trie.link[p]
		delta = h - r
		for trie.ch[r+delta] == empty && trie.sibling[r] != trie.link[p] {
			r = trie.sibling[r]
		}
		if trie.ch[r+delta] == empty {
			break // found a slot
		}
	}
	//if int(h) == lasth {
	if trys == tolerance {
		T().Errorf("abort find")
		panic("abort find")
	}
	return p, q
}

func (trie *ShortHashTrie) Iterator() *TrieIterator {
	iterator := &TrieIterator{
		trie: trie,
		n:    0,
	}
	return iterator
}

// --- Iterator --------------------------------------------------------------

type TrieIterator struct {
	trie     *ShortHashTrie
	position pointer
	n        int
}

func (ti *TrieIterator) Next(c int8) int {
	ti.position = ti.trie.advanceToChild(ti.position, bidiclass(c), ti.n)
	T().Debugf("advanced to p=%d with c=%d", ti.position, c)
	return int(ti.position)
}
