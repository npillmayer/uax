/*
Package for a generator for UAX#29 Grapheme classes.

Contents

Generator for Unicode UAX#29 grapheme code-point classes.
For more information see https://unicode.org/reports/tr29/.

Classes are generated from a companion file: "GraphemeBreakProperty.txt".
This is the definite source for UAX#29 code-point classes.

This creates a file "graphemeclasses.go" in the current directory. It is designed
to be called from the "grapheme" directory.


License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package main

import (
	"bytes"
	"flag"
	"go/format"
	"io/ioutil"
	"log"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/npillmayer/uax/internal/testdata"
	"github.com/npillmayer/uax/internal/ucdparse"
	"golang.org/x/text/unicode/rangetable"
)

func main() {
	flag.Parse()

	codePointLists, err := loadUnicodeGraphemeBreakFile()
	checkFatal(err)

	classes := []string{}
	for class := range codePointLists {
		classes = append(classes, class)
	}
	sort.Strings(classes)

	var w bytes.Buffer
	terr := T.Execute(&w, map[string]interface{}{
		"Classes":    classes,
		"Codepoints": codePointLists,
	})
	checkFatal(terr)

	formatted, err := format.Source(w.Bytes())
	checkFatal(err)

	err = ioutil.WriteFile("graphemeclasses.go", formatted, 0644)
	checkFatal(err)
}

// Load the Unicode UAX#29 definition file: GraphemeBreakProperty.txt
func loadUnicodeGraphemeBreakFile() (map[string][]rune, error) {
	parser, err := ucdparse.New(bytes.NewReader(testdata.GraphemeBreakProperty))
	runeranges := map[string][]rune{}
	for parser.Next() {
		from, to := parser.Token.Range()
		clstr := strings.TrimSpace(parser.Token.Field(1))
		if clstr == "" {
			// Not quite sure why this happens.
			log.Println("found empty class")
			continue
		}
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

var T = template.Must(template.New("").Funcs(template.FuncMap{
	"rangetable": func(runes []rune) *unicode.RangeTable {
		return rangetable.New(runes...)
	},
}).Parse(`package grapheme

// Code generated by github.com/npillmayer/uax/grapheme/internal/generator  DO NOT EDIT
//
// BSD License, Copyright (c) 2018, Norbert Pillmayer (norbert@pillmayer.com)

import (
    "strconv"
    "unicode"
)

// Type for UAX#29 grapheme classes.
// Must be convertable to int.
type GraphemeClass int

// These are all the UAX#29 breaking classes.
const (
{{ range $i, $class := .Classes }}
	{{$class}}Class GraphemeClass = {{$i}}
{{- end }}

	Any GraphemeClass = 999
    sot GraphemeClass = 1000 // pseudo class "start of text"
    eot GraphemeClass = 1001 // pseudo class "end of text"
)

// Range tables for UAX#29 grapheme classes.
// Clients can check with unicode.Is(..., rune)
var (
{{ range $i, $class := .Classes }}
	{{$class}} = _{{$class}}
{{- end }}
)

// Stringer for type GraphemeClass
func (c GraphemeClass) String() string {
	switch c {
	case sot: return "sot"
	case eot: return "eot"
	case Any: return "Any"
	default:
		return "GraphemeClass(" + strconv.Itoa(int(c)) + ")"
{{- range $i, $class := .Classes }}
	case {{$class}}Class: return "{{ $class }}Class"
{{- end }}
	}
}

var rangeFromGraphemeClass = []*unicode.RangeTable{
{{- range $i, $class := .Classes }}
	{{$class}}Class: {{$class}},
{{- end }}
}


// range table definitions, these are separate from the public definitions
// to make documentation readable.
var (
{{- range $class, $runes := .Codepoints }}
	_{{ $class }} = {{ printf "%#v" (rangetable $runes) }}
{{- end }}
)
`))

// --- Util -------------------------------------------------------------

func checkFatal(err error) {
	_, file, line, _ := runtime.Caller(1)
	if err != nil {
		log.Fatalln(":", file, ":", line, "-", err)
	}
}
