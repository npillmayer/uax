package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/uax/internal/ucdparse"
)

func main() {
	tlevel := flag.String("trace", "I", "Trace level")
	outf := flag.String("o", "_bracketpairs.go", "Output file name")
	pkg := flag.String("pkg", "main", "Package name to use in output file")
	flag.Parse()
	logAdapter := gologadapter.GetAdapter()
	trace := logAdapter()
	trace.SetTraceLevel(traceLevel(*tlevel))
	tracing.SetTraceSelector(mytrace{tracer: trace})
	tracing.Infof("Generating Unicode bracket pairs")
	pairs := readBrackets()
	tracing.Infof("Read %d bracket pairs", len(pairs))
	if len(pairs) == 0 {
		tracing.Errorf("Did not read any bracket pairs, exiting")
		os.Exit(1)
	}
	f, err := os.Create(*outf)
	if err != nil {
		tracing.Errorf(err.Error())
		os.Exit(2)
	}
	defer f.Close()
	f.WriteString("package " + *pkg + "\n\n")
	f.WriteString("type BracketPair struct {\n    o rune\n    c rune\n}\n\n")
	f.WriteString("var UAX9BracketPairs = []BracketPair{\n")
	for _, p := range pairs {
		//f.WriteString(fmt.Sprintf("    BracketPair{o: %q, c: %q},\n", p.o, p.c))
		f.WriteString(fmt.Sprintf("    {o: %q, c: %q},\n", p.o, p.c))
	}
	f.WriteString("}\n")
}

type bracketPair struct {
	o rune
	c rune
}

func readBrackets() []bracketPair {
	file, err := os.Open("./BidiBrackets.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()
	tracing.Infof("Found file BidiBrackets.txt ...")
	bracketList := make([]bracketPair, 0, 65)
	err = ucdparse.Parse(file, func(t *ucdparse.Token) {
		if typ := strings.TrimSpace(t.Field(2)); typ != "o" {
			return
		}
		pair := bracketPair{}
		pair.o, _ = t.Range()
		pair.c = readHexRune(t.Field(1))
		bracketList = append(bracketList, pair)
		tracing.Debugf(t.Comment)
	})
	if err != nil {
		tracing.Errorf(err.Error())
		os.Exit(1)
	}
	tracing.Debugf("done.")
	return bracketList
}

func readHexRune(inp string) rune {
	inp = strings.TrimSpace(inp)
	n, _ := strconv.ParseUint(inp, 16, 64)
	return rune(n)
}

func traceLevel(l string) tracing.TraceLevel {
	switch l {
	case "D":
		return tracing.LevelDebug
	case "I":
		return tracing.LevelInfo
	case "E":
		return tracing.LevelError
	}
	return tracing.LevelDebug
}

type mytrace struct {
	tracer tracing.Trace
}

func (t mytrace) Select(string) tracing.Trace {
	return t.tracer
}
