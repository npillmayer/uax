package segment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestWhitespace1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter()
	seg.Init(strings.NewReader("Hello World!"))
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment = '%s' with p = %d|%d", seg.Text(), p1, p2)
	}
}

func TestWhitespace2(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	seg := NewSegmenter()
	seg.Init(strings.NewReader("	for (i=0; i<5; i++)   count += i;"))
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment = '%s' with p = %d|%d", seg.Text(), p1, p2)
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
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		n++
	}
	if n != 4 {
		t.Errorf("Expected 4 segments, have %d", n)
	}
}
func TestSimpleSegmenter2(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("lime-tree"))
	n := 0
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		n++
	}
	if n != 1 {
		t.Errorf("Expected 1 segment, have %d", n)
	}
}

func TestBounded(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	seg := NewSegmenter(NewSimpleWordBreaker())
	seg.Init(strings.NewReader("Hello World, how are you?"))
	n := 0
	output := ""
	for seg.BoundedNext(14) {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		output += " [" + seg.Text() + "]"
		n++
	}
	t.Logf("seg.Err() = %v", seg.Err())
	t.Logf("seg.Text() = '%s'", seg.Text())
	t.Logf("bounded: output = %v", output)
	if n != 5 {
		t.Fatalf("Expected 5 segments, have %d", n)
	}
	t.Logf("bounded: passed 1st test ")
	tracer().Infof("======= rest =======")
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		output += " [" + seg.Text() + "]"
		n++
	}
	t.Logf("output = %v", output)
	if n != 10 {
		t.Errorf("Expected 10 segments, have %d", n)
	}
}

func TestSimpleSegnew(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	seg := NewSegmenter(NewSimpleWordBreaker())
	seg.Init(strings.NewReader("Hello World, how are you?"))
	n := 0
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("********* segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
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
		p1, p2 := seg.Penalties()
		fmt.Printf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
	}
	// Output:
	// segment: penalty =   100|0 for breaking after 'Hello'
	// segment: penalty =  -100|0 for breaking after ' '
	// segment: penalty =   100|0 for breaking after 'World!'
}
