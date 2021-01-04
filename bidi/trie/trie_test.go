package trie

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestEnterSimple(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	//trie := NewShortHashTrie(9973, 46)
	trie := NewShortHashTrie(9973, 46)
	p := trie.findPositionForWord([]byte{13, 20})
	t.Logf("p=%d", p)
	p = trie.findPositionForWord([]byte{13, 21})
	t.Logf("p=%d", p)
	p = trie.findPositionForWord([]byte{13, 21, 10})
	t.Logf("p=%d", p)
	t.Fail()
}

func TestMakeTable(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	for l := len(primes) - 1; l > 0; l-- {
		size := primes[l]
		trie := NewShortHashTrie(int16(size), 46)
		p := trie.findPositionForWord([]byte{13, 20})
		t.Logf("p=%d", p) // TODO insert all bidi rules
		// and wait for panic to happen
	}
}
