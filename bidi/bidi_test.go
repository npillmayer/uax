package bidi

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/text/unicode/bidi"

	"github.com/npillmayer/schuko/tracing/gologadapter"
)

func TestClasses(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	//teardown := gologadapter.RedirectTracing(t)
	//defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	t.Logf("L = %s", ClassString(bidi.L))
	t.Logf("NI = %s", ClassString(NI))
	t.Logf("BRACKC = %s", ClassString(BRACKC))
	t.Logf("MAX = %s", ClassString(MAX))
}

func TestSimple(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	//teardown := gologadapter.RedirectTracing(t)
	//defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("Hello 123.456")
	ordering := Parse(reader)
	t.Logf("resulting ordering = %s", ordering)
	t.Fail()
}
