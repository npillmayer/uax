/*
Package for a generator for UAX#29 word breaking classes.

BSD License

Copyright (c) 2017–20, Norbert Pillmayer (norbert@pillmayer.com)


Contents

This is a generator for Unicode UAX#29 word breaking code-point classes.
For more information see http://unicode.org/reports/tr29/

Classes are generated from a UAX#29 companion file: "WordBreakProberty.txt".
This is the definite source for UAX#29 code-point classes.


This creates a file "uax29classes.go" in the current directory. It is designed
to be called from the "uax29" directory.


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

	codePointLists, err := loadUnicodeLineBreakFile()
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

	err = ioutil.WriteFile("uax29classes.go", formatted, 0644)
	checkFatal(err)
}

// Load the Unicode UAX#29 definition file: WordBreakProperty.txt
func loadUnicodeLineBreakFile() (map[string][]rune, error) {
	p, err := ucdparse.New(bytes.NewReader(testdata.WordBreakProperty))
	if err != nil {
		return nil, err
	}

	runeranges := map[string][]rune{}
	for p.Next() {
		from, to := p.Token.Range()
		brclzstr := strings.TrimSpace(p.Token.Field(1))
		if brclzstr == "" {
			// Not quite sure why this happens.
			log.Println("found empty class")
			continue
		}
		list := runeranges[brclzstr]
		for r := from; r <= to; r++ {
			list = append(list, r)
		}
		runeranges[brclzstr] = list
	}
	err = p.Token.Error
	if err != nil {
		log.Fatal(err)
	}
	return runeranges, err
}

var T = template.Must(template.New("").Funcs(template.FuncMap{
	"rangetable": func(runes []rune) *unicode.RangeTable {
		return rangetable.New(runes...)
	},
}).Parse(`package uax29

// Code generated by github.com/npillmayer/uax/uax29/internal/generator  DO NOT EDIT
//
// BSD License, Copyright (c) 2018, Norbert Pillmayer (norbert@pillmayer.com)

import (
    "strconv"
    "unicode"
)

// Type for UAX#29 code-point classes.
// Must be convertable to int.
type UAX29Class int

// These are all the UAX#29 breaking classes.
const (
{{ range $i, $class := .Classes }}
	{{$class}}Class UAX29Class = {{$i}}
{{- end }}

	Other UAX29Class = 999
    sot UAX29Class = 1000 // pseudo class "start of text"
    eot UAX29Class = 1001 // pseudo class "end of text"
)

// Range tables for UAX#29 code-point classes.
// Clients can check with unicode.Is(..., rune)
var (
{{ range $i, $class := .Classes }}
	{{$class}} = _{{$class}}
{{- end }}
)

// Stringer for type UAX29Class
func (c UAX29Class) String() string {
	switch c {
	case sot: return "sot"
	case eot: return "eot"
	case Other: return "Other"
	default:
		return "UAX29Class(" + strconv.Itoa(int(c)) + ")"
{{- range $i, $class := .Classes }}
	case {{$class}}Class: return "{{ $class }}Class"
{{- end }}
	}
}

var rangeFromUAX29Class = []*unicode.RangeTable{
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
