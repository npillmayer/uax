package bidi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/text/unicode/bidi"

	//"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestClasses(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	t.Logf("L = %s", ClassString(bidi.L))
	if ClassString(bidi.L) != "L" {
		t.Errorf("string for L not as expected")
	}
	t.Logf("NI = %s", ClassString(NI))
	if ClassString(NI) != "NI" {
		t.Errorf("string for NI not as expected")
	}
	t.Logf("BRACKC = %s", ClassString(BRACKC))
	if ClassString(BRACKC) != "BRACKC" {
		t.Errorf("string for BRACKC not as expected")
	}
}

func TestSimple(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("Hello 123.456")
	ordering := ResolveParagraph(reader)
	fmt.Printf("resulting ordering = %s\n", ordering)
	//t.Fail()
}
