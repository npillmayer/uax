package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/npillmayer/uax/internal/ucdparse"
)

func main() {
	tlevel := flag.String("trace", "I", "Trace level")
	outf := flag.String("o", "_bracketpairs.go", "Output file name")
	pkg := flag.String("pkg", "main", "Package name to use in output file")
	flag.Parse()

	if *tlevel == "D" {
		debugEnabled = true
	}

	Infof("Generating Unicode bracket pairs")
	pairs := readBrackets()
	Infof("Read %d bracket pairs", len(pairs))
	if len(pairs) == 0 {
		Errorf("Did not read any bracket pairs, exiting")
		os.Exit(1)
	}
	f, err := os.Create(*outf)
	if err != nil {
		Errorf(err.Error())
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
	Infof("Found file BidiBrackets.txt ...")
	bracketList := make([]bracketPair, 0, 65)
	err = ucdparse.Parse(file, func(t *ucdparse.Token) {
		if typ := strings.TrimSpace(t.Field(2)); typ != "o" {
			return
		}
		pair := bracketPair{}
		pair.o, _ = t.Range()
		pair.c = readHexRune(t.Field(1))
		bracketList = append(bracketList, pair)
		Debugf(t.Comment)
	})
	if err != nil {
		Errorf(err.Error())
		os.Exit(1)
	}
	Debugf("done.")
	return bracketList
}

func readHexRune(inp string) rune {
	inp = strings.TrimSpace(inp)
	n, _ := strconv.ParseUint(inp, 16, 64)
	return rune(n)
}

var debugEnabled bool

func Debugf(format string, args ...interface{}) {
	printf(os.Stdout, format, args...)
}

func Infof(format string, args ...interface{}) {
	printf(os.Stdout, format, args...)
}

func Errorf(format string, args ...interface{}) {
	printf(os.Stderr, format, args...)
}

func printf(out io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(out, format, args...)
	if strings.HasSuffix(format, "\n") {
		out.Write([]byte{'\n'})
	}
}
