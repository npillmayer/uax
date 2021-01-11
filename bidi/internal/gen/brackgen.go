package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/uax/ucd"
)

// T traces to the global core tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}

func main() {
	gtrace.CoreTracer = gologadapter.New()
	tlevel := flag.String("trace", "I", "Trace level")
	outf := flag.String("o", "_bracketpairs.go", "Output file name")
	pkg := flag.String("pkg", "main", "Package name to use in output file")
	flag.Parse()
	T().Infof("Generating Unicode bracket pairs")
	T().SetTraceLevel(traceLevel(*tlevel))
	pairs := readBrackets()
	T().Infof("Read %d bracket pairs", len(pairs))
	if len(pairs) == 0 {
		T().Errorf("Did not read any bracket pairs, exiting")
		os.Exit(1)
	}
	f, err := os.Create(*outf)
	if err != nil {
		T().Errorf(err.Error())
		os.Exit(2)
	}
	defer f.Close()
	f.WriteString("package " + *pkg + "\n\n")
	f.WriteString("type BracketPair struct {\n    o rune\n    c rune\n}\n\n")
	f.WriteString("var UAX9BracketPairs = []BracketPair{\n")
	for _, p := range pairs {
		f.WriteString(fmt.Sprintf("    BracketPair{o: %q, c: %q},\n", p.o, p.c))
	}
	f.WriteString("}\n")
}

type bracketPair struct {
	o rune
	c rune
}

func readBrackets() []bracketPair {
	tf := ucd.OpenTestFile("./BidiBrackets.txt", nil)
	T().Infof("Found file BidiBrackets.txt ...")
	defer tf.Close()
	bracketList := make([]bracketPair, 0, 65)
	for tf.Scan() {
		fields := strings.Split(tf.Text(), ";")
		if len(fields) >= 3 {
			typ := strings.TrimSpace(fields[2])
			if typ != "o" {
				continue
			}
			pair := bracketPair{}
			pair.o = readHexRune(fields[0])
			pair.c = readHexRune(fields[1])
			bracketList = append(bracketList, pair)
			T().Debugf(strings.TrimSpace(tf.Comment()))
		}
	}
	T().Debugf("done.")
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
