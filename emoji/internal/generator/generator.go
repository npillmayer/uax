/*
Package for a generator for UTS#51 Emoji character classes.

Content

Generator for Unicode Emoji code-point classes. For more information
see http://www.unicode.org/reports/tr51/#Emoji_Properties_and_Data_Files

Classes are generated from a companion file: "emoji-data.txt".


Usage

The generator has just one option, a "verbose" flag. It should usually
be turned on.

   generator [-v]

This creates a file "emojiclasses.go" in the current directory. It is designed
to be called from the "emoji" directory.


License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"text/template"
	"time"

	"os"

	"github.com/npillmayer/uax/internal/testdata"
	"github.com/npillmayer/uax/internal/ucdparse"
)

var logger = log.New(os.Stderr, "UTS#51 generator: ", log.LstdFlags)

// flag: verbose output ?
var verbose bool

var emojiClassnames = []string{
	"Emoji",
	"Emoji_Presentation",
	"Emoji_Modifier",
	"Emoji_Modifier_Base",
	"Emoji_Component",
	"Extended_Pictographic",
}

// Load the Unicode UAX#29 definition file: EmojiBreakProperty.txt
func loadUnicodeEmojiBreakFile() (map[string][]rune, error) {
	if verbose {
		logger.Printf("reading EmojiBreakProperty.txt")
	}
	defer timeTrack(time.Now(), "loading emoji-data.txt")

	parser, err := ucdparse.New(bytes.NewReader(testdata.EmojiBreakProperty))
	if err != nil {
		return nil, err
	}
	runeranges := make(map[string][]rune, len(emojiClassnames))
	for parser.Next() {
		from, to := parser.Token.Range()
		clstr := strings.TrimSpace(parser.Token.Field(1))
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

var header = `package emoji

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
// Type for UTS#51 emoji code-point classes.
// Must be convertable to int.
type EmojisClass int

// Will be initialized in SetupEmojisClasses()
var rangeFromEmojisClass []*unicode.RangeTable
`

var templateRangeTableVars = `
// Range tables for emoji code-point classes.
// Will be initialized with SetupEmojisClasses().
// Clients can check with unicode.Is(..., rune){{$i:=0}}
var {{range .}}{{$i = inc $i}}{{.}}, {{if modeight $i}}
    {{end}}{{end}}unused *unicode.RangeTable
`

var templateClassConsts = `
// These are all the emoji breaking classes.
const ( {{$i:=0}}
{{range  .}}    {{.}}Class EmojisClass = {{$i}}{{$i = inc $i}}
{{end}})
`

//{{range  $k,$v := .}}    {{$k}}Class EmojisClass = {{$v}}

var templateClassStringer = `
const _EmojisClass_name = "{{range $c,$name := .}}{{$name}}Class{{end}}"

var _EmojisClass_index = [...]uint16{0{{startinxs .}} }

// Stringer for type EmojisClass
func (c EmojisClass) String() string {
    if c < 0 || c >= EmojisClass(len(_EmojisClass_index)-1) {
        return "EmojisClass(" + strconv.FormatInt(int64(c), 10) + ")"
    }
    return _EmojisClass_name[_EmojisClass_index[c]:_EmojisClass_index[c+1]]
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
	w.WriteString("\nfunc setupEmojisClasses() {\n")
	w.WriteString("    rangeFromEmojisClass = make([]*unicode.RangeTable, int(Extended_PictographicClass)+1)\n")
	t := makeTemplate("Emoji range", templateRangeForClass)

	lastWriteOrder := []string{
		"Emoji",
		"Emoji_Presentation",
		"Emoji_Modifier",
		"Emoji_Modifier_Base",
		"Emoji_Component",
		"Extended_Pictographic",
	}

	for _, key := range lastWriteOrder {
		codepoints := codePointLists[key]
		w.WriteString(fmt.Sprintf("\n    // Range for Emoji class %s\n", key))
		w.WriteString(fmt.Sprintf("    %s = rangetable.New(", key))
		checkFatal(t.Execute(w, codepoints))
		w.WriteString(")\n")
		w.WriteString(fmt.Sprintf("    rangeFromEmojisClass[int(%sClass)] = %s\n", key, key))
	}
	w.WriteString("}\n")
}

func main() {
	doVerbose := flag.Bool("v", false, "verbose output mode")
	flag.Parse()
	verbose = *doVerbose
	codePointLists, err := loadUnicodeEmojiBreakFile()
	checkFatal(err)
	if verbose {
		logger.Printf("loaded %d Emoji breaking classes\n", len(codePointLists))
	}
	f, ioerr := os.Create("emojiclasses.go")
	checkFatal(ioerr)
	defer f.Close()
	w := bufio.NewWriter(f)
	w.WriteString(header)
	w.WriteString(templateClassType)
	t := makeTemplate("Emoji classes", templateClassConsts)
	checkFatal(t.Execute(w, emojiClassnames))
	t = makeTemplate("Emoji range tables", templateRangeTableVars)
	checkFatal(t.Execute(w, emojiClassnames))
	t = makeTemplate("Emoji classes stringer", templateClassStringer)
	checkFatal(t.Execute(w, emojiClassnames))
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
