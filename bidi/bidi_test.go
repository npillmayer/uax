package bidi

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"

	"github.com/npillmayer/schuko/tracing/gologadapter"
)

func TestSimple(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	//teardown := gologadapter.RedirectTracing(t)
	//defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	reader := strings.NewReader("123.456")
	ordering := Parse(reader)
	t.Logf("resulting ordering = %s", ordering)
	t.Fail()
}
