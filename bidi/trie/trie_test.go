package trie

import (
	"testing"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestEnterSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "uax.bidi")
	defer teardown()
	tracing.Select("uax.bidi").SetTraceLevel(tracing.LevelError)
	//
	trie, _ := NewTinyHashTrie(139, 46)
	p1 := trie.AllocPositionForWord([]byte{13, 20})
	t.Logf("p=%d", p1)
	p2 := trie.AllocPositionForWord([]byte{13, 21})
	t.Logf("p=%d", p2)
	if p2 != p1+1 {
		t.Errorf("expected to be p1 and p2 consecutive, aren't: %d / %d", p1, p2)
	}
	p3 := trie.AllocPositionForWord([]byte{13, 21, 10})
	t.Logf("p=%d", p3)
	if p3 == p2 {
		t.Errorf("expected p2 and p3 to be different, aren't")
	}
}

func TestEnterZero(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "uax.bidi")
	defer teardown()
	tracing.Select("uax.bidi").SetTraceLevel(tracing.LevelError)
	//
	trie, _ := NewTinyHashTrie(139, 46)
	p1 := trie.AllocPositionForWord([]byte{0, 0})
	t.Logf("p=%d", p1)
	if p1 == 0 {
		t.Errorf("no entry for [0,0]")
	}
}

func TestIterator(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "uax.bidi")
	defer teardown()
	tracing.Select("uax.bidi").SetTraceLevel(tracing.LevelError)
	//
	trie, _ := NewTinyHashTrie(139, 46)
	word := []byte{13, 20}
	p := trie.AllocPositionForWord(word)
	t.Logf("p=%d\nFreeze----", p)
	trie.Freeze()
	q := trie.AllocPositionForWord(word)
	t.Logf("lookup p=%d", q)
	if p != q {
		t.Fatalf("expected to find word again in trie, couldn't: %d != %d", q, p)
	}
	it := trie.Iterator()
	for i, w := range word {
		p = it.Next(int8(w))
		if p == 0 {
			t.Errorf("no position for byte #%d=%v", i, w)
		}
		t.Logf("p=%d", p)
	}
	if p != q {
		t.Fatalf("expected to iterate to word in trie, couldn't: %d != %d", q, p)
	}
}
