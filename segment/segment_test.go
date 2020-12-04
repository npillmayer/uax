package segment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestWhitespace1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter()
	seg.Init(strings.NewReader("Hello World!"))
	for seg.Next() {
		t.Logf("segment = '%s' with p = %d", seg.Text(), seg.Penalties()[0])
	}
}

func TestWhitespace2(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter()
	seg.Init(strings.NewReader("	for (i=0; i<5; i++)   count += i;"))
	for seg.Next() {
		t.Logf("segment = '%s' with p = %d", seg.Text(), seg.Penalties()[0])
	}
}

func TestSimpleSegmenter1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("Hello World "))
	n := 0
	for seg.Next() {
		t.Logf("segment: penalty = %5d for breaking after '%s'\n", seg.Penalties()[0], seg.Text())
		n++
	}
	if n != 4 {
		t.Errorf("Expected 4 segments, have %d", n)
	}
}
func TestSimpleSegmenter2(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("lime-tree"))
	n := 0
	for seg.Next() {
		t.Logf("segment: penalty = %5d for breaking after '%s'\n", seg.Penalties()[0], seg.Text())
		n++
	}
	if n != 1 {
		t.Errorf("Expected 1 segment, have %d", n)
	}
}

func TestSimpleSegmenter3(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("Hello World, how are you?"))
	n := 0
	for seg.Next() {
		t.Logf("segment: penalty = %5d for breaking after '%s'\n", seg.Penalties()[0], seg.Text())
		n++
	}
	if n != 9 {
		t.Errorf("Expected 9 segments, have %d", n)
	}
}

func ExampleSegmenter() {
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("Hello World!"))
	for seg.Next() {
		fmt.Printf("segment: penalty = %5d for breaking after '%s'\n", seg.Penalties()[0], seg.Text())
	}
	// Output:
	// segment: penalty =   100 for breaking after 'Hello'
	// segment: penalty =  -100 for breaking after ' '
	// segment: penalty = -1000 for breaking after 'World!'
}
