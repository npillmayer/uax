/*
Package for a generator for UAX#29 Grapheme classes.

Contents

Generator for Unicode UAX#29 grapheme code-point classes.
For more information see https://unicode.org/reports/tr29/.

Classes are generated from a companion file: "GraphemeBreakProperty.txt".
This is the definite source for UAX#29 code-point classes. The
generator looks for it in a directory "$GOPATH/etc/".


Usage

The generator has just one option, a "verbose" flag. It should usually
be turned on.

   generator [-v]

This creates a file "graphemeclasses.go" in the current directory. It is designed
to be called from the "grapheme" directory.


License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"runtime"
	"text/template"
	"time"

	"os"

	"github.com/npillmayer/uax/internal/ucdparse"
)

var logger = log.New(os.Stderr, "UAX#29 generator: ", log.LstdFlags)

// flag: verbose output ?
var verbose bool

var graphemeClassnames = []string{
	"CR",
	"LF",
	"Prepend",
	"Control",
	"Extend",
	"Regional_Indicator",
	"SpacingMark",
	"L",
	"V",
	"T",
	"LV",
	"LVT",
	"ZWJ",
}

// Load the Unicode UAX#29 definition file: GraphemeBreakProperty.txt
func loadUnicodeGraphemeBreakFile() (map[string][]rune, error) {
	if verbose {
		logger.Printf("reading GraphemeBreakProperty.txt")
	}
	defer timeTrack(time.Now(), "loading GraphemeBreakProperty.txt")
	gopath := os.Getenv("GOPATH")
	f, err := os.Open(gopath + "/etc/GraphemeBreakProperty.txt")
	if err != nil {
		fmt.Printf("ERROR loading " + gopath + "/etc/GraphemeBreakProperty.txt\n")
		return nil, err
	}
	defer f.Close()
	parser, err := ucdparse.New(f)
	runeranges := make(map[string][]rune, len(graphemeClassnames))
	for parser.Next() {
		from, to := parser.Token.Range()
		clstr := parser.Token.Field(1)
		list := runeranges[clstr]
		for r := from; r <= to; r++ {
			list = append(list, r)
		}
		runeranges[clstr] = list
	}
	err = parser.Token.Error
	if err != nil {
		log.Fatal(err)
	}
	return runeranges, err
}

// --- Templates --------------------------------------------------------

var header = `package grapheme

// This file has been generated -- you probably should NOT EDIT IT !
// 
// BSD License, Copyright (c) 2018, Norbert Pillmayer (norbert@pillmayer.com)

import (
    "strconv"
    "unicode"

    "golang.org/x/text/unicode/rangetable"
)
`

var templateClassType = `
// Type for UAX#29 grapheme code-point classes.
// Must be convertable to int.
type GraphemeClass int

// Will be initialized in SetupGraphemeClasses()
var rangeFromGraphemeClass []*unicode.RangeTable
`

var templateRangeTableVars = `
// Range tables for grapheme code-point classes.
// Will be initialized with SetupGraphemeClasses().
// Clients can check with unicode.Is(..., rune){{$i:=0}}
var {{range .}}{{$i = inc $i}}{{.}}, {{if modten $i}}
    {{end}}{{end}}unused *unicode.RangeTable
`

var templateClassConsts = `
// These are all the grapheme breaking classes.
const ( {{$i:=0}}
{{range  .}}    {{.}}Class GraphemeClass = {{$i}}{{$i = inc $i}}
{{end}}
	Any GraphemeClass = 999
    sot GraphemeClass = 1000 // pseudo class "start of text"
    eot GraphemeClass = 1001 // pseudo class "end of text"
)
`

//{{range  $k,$v := .}}    {{$k}}Class GraphemeClass = {{$v}}

var templateClassStringer = `
const _GraphemeClass_name = "{{range $c,$name := .}}{{$name}}Class{{end}}"

var _GraphemeClass_index = [...]uint16{0{{startinxs .}} }

// Stringer for type GraphemeClass
func (c GraphemeClass) String() string {
    if c == sot {
        return "sot"
    } else if c == eot {
        return "eot"
    } else if c == Any {
    	return "Any"
    } else if c < 0 || c >= GraphemeClass(len(_GraphemeClass_index)-1) {
        return "GraphemeClass(" + strconv.FormatInt(int64(c), 10) + ")"
    }
    return _GraphemeClass_name[_GraphemeClass_index[c]:_GraphemeClass_index[c+1]]
}
`

var templateRangeForClass = `{{$i:=0}}{{range .}}{{if notfirst $i}}, {{if modeight $i}}
    {{end}}{{end}}{{$i = inc $i}}{{printf "%+q" .}}{{end}}`

// Helper functions for templates
var funcMap = template.FuncMap{
	"modten": func(i int) bool {
		return i%10 == 0
	},
	"modeight": func(i int) bool {
		return (i+2)%8 == 0
	},
	"inc": func(i int) int {
		return i + 1
	},
	"notfirst": func(i int) bool {
		return i > 0
	},
	"startinxs": func(str []string) string {
		out := ""
		total := 0
		for _, s := range str {
			l := len(s) + 5
			total += l
			if (41+len(out))%80 > 75 {
				out += fmt.Sprintf(",\n    %d", total)
			} else {
				out += fmt.Sprintf(", %d", total)
			}
		}
		return out
	},
}

func makeTemplate(name string, templString string) *template.Template {
	if verbose {
		logger.Printf("creating %s", name)
	}
	t := template.Must(template.New(name).Funcs(funcMap).Parse(templString))
	return t
}

// --- Main -------------------------------------------------------------

func generateRanges(w *bufio.Writer, codePointLists map[string][]rune) {
	defer timeTrack(time.Now(), "generate range tables")
	w.WriteString("\nfunc setupGraphemeClasses() {\n")
	w.WriteString("    rangeFromGraphemeClass = make([]*unicode.RangeTable, int(ZWJClass)+1)\n")
	t := makeTemplate("Grapheme range", templateRangeForClass)
	for key, codepoints := range codePointLists {
		w.WriteString(fmt.Sprintf("\n    // Range for Grapheme class %s\n", key))
		w.WriteString(fmt.Sprintf("    %s = rangetable.New(", key))
		checkFatal(t.Execute(w, codepoints))
		w.WriteString(")\n")
		w.WriteString(fmt.Sprintf("    rangeFromGraphemeClass[int(%sClass)] = %s\n", key, key))
	}
	w.WriteString("}\n")
}

func main() {
	doVerbose := flag.Bool("v", false, "verbose output mode")
	flag.Parse()
	verbose = *doVerbose
	codePointLists, err := loadUnicodeGraphemeBreakFile()
	checkFatal(err)
	if verbose {
		logger.Printf("loaded %d Grapheme breaking classes\n", len(codePointLists))
	}
	f, ioerr := os.Create("graphemeclasses.go")
	checkFatal(ioerr)
	defer f.Close()
	w := bufio.NewWriter(f)
	w.WriteString(header)
	w.WriteString(templateClassType)
	t := makeTemplate("Grapheme classes", templateClassConsts)
	checkFatal(t.Execute(w, graphemeClassnames))
	t = makeTemplate("Grapheme range tables", templateRangeTableVars)
	checkFatal(t.Execute(w, graphemeClassnames))
	t = makeTemplate("Grapheme classes stringer", templateClassStringer)
	checkFatal(t.Execute(w, graphemeClassnames))
	generateRanges(w, codePointLists)
	w.Flush()
}

// --- Util -------------------------------------------------------------

// Little helper for testing
func timeTrack(start time.Time, name string) {
	if verbose {
		elapsed := time.Since(start)
		logger.Printf("timing: %s took %s\n", name, elapsed)
	}
}

func checkFatal(err error) {
	_, file, line, _ := runtime.Caller(1)
	if err != nil {
		logger.Fatalln(":", file, ":", line, "-", err)
	}
}
