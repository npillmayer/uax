/*
Package ucd provides a parser for Unicode Character Database files.

Package ucd provides a parser for Unicode Character Database files, the
format of which is defined in http://www.unicode.org/reports/tr44/. See
http://www.unicode.org/Public/UCD/latest/ucd/ for example files.

This is an adaption of a package from the x/text/... tree of the
Go standard library. Unfortunately, the ucd package there is "internal" and
thus may not be imported. We replicate it here (my understanding is that
this is legally ok), but taylor it slightly
to our needs. We keep it as an internal package as well, as it is
used for table-generation and for tests only. No application source
code references it.

License

Copyright 2014 The Go Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the GO-LICENSE file in directory uax,
the base directory of this module.

___________________________________________________________________________

Parts of this package not governed by the Go license are subject to the
following license:

Copyright © 2021, Norbert Pillmayer, all rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. */
package ucd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Parse calls f for each entry in the given reader of a UCD file. It will close
// the reader upon return. It will call log.Fatal if any error occurred.
//
// This implements the most common usage pattern of using Parser.
func Parse(r io.ReadCloser, f func(p *Parser), options ...Option) {
	defer r.Close()

	p := NewUCDParser(r)
	for _, o := range options {
		o(p)
	}
	for p.Next() {
		f(p)
	}
	if err := p.Err(); err != nil {
		r.Close() // os.Exit will cause defers not to be called.
		log.Fatal(err)
	}
}

// A Parser parses Unicode Character Database (UCD) files.
type Parser struct {
	scanner *bufio.Scanner

	keepRanges bool // Don't expand rune ranges in field 0.

	err     error
	comment string
	field   []string
	// parsedRange is needed in case Range(0) is called more than once for one
	// field. In some cases this requires scanning ahead.
	line                 int
	parsedRange          bool
	rangeStart, rangeEnd rune

	partHandler    func(p *Parser)
	commentHandler func(s string)
}

func (p *Parser) setError(err error, msg string) {
	if p.err == nil && err != nil {
		if msg == "" {
			p.err = fmt.Errorf("ucd:line:%d: %v", p.line, err)
		} else {
			p.err = fmt.Errorf("ucd:line:%d:%s: %v", p.line, msg, err)
		}
	}
}

func (p *Parser) getField(i int) string {
	if i >= len(p.field) {
		return ""
	}
	return p.field[i]
}

// Err returns a non-nil error if any error occurred during parsing.
func (p *Parser) Err() error {
	return p.err
}

// NewUCDParser returns a Parser for the given Reader.
func NewUCDParser(r io.Reader) *Parser {
	p := &Parser{
		scanner: bufio.NewScanner(r),
	}
	return p
}

// Next parses the next line in the file. It returns true if a line was parsed
// and false if it reached the end of the file.
func (p *Parser) Next() bool {
	if !p.keepRanges && p.rangeStart < p.rangeEnd {
		p.rangeStart++
		return true
	}
	p.comment = ""
	p.field = p.field[:0]
	p.parsedRange = false

	for p.scanner.Scan() && p.err == nil {
		p.line++
		s := p.scanner.Text()
		//fmt.Printf("new line = %s\n", s)
		if s == "" {
			continue
		}
		if s[0] == '#' {
			if p.commentHandler != nil {
				p.commentHandler(strings.TrimSpace(s[1:]))
			}
			continue
		}

		// Parse line
		if i := strings.IndexByte(s, '#'); i != -1 {
			p.comment = strings.TrimSpace(s[i+1:])
			s = s[:i]
		}
		if s[0] == '@' {
			if p.partHandler != nil {
				p.field = append(p.field, strings.TrimSpace(s[1:]))
				p.partHandler(p)
				p.field = p.field[:0]
			}
			p.comment = ""
			continue
		}
		for {
			i := strings.IndexByte(s, ';')
			if i == -1 {
				p.field = append(p.field, strings.TrimSpace(s))
				break
			}
			p.field = append(p.field, strings.TrimSpace(s[:i]))
			s = s[i+1:]
		}
		if !p.keepRanges {
			p.rangeStart, p.rangeEnd = p.getRange(0)
		}
		return true
	}
	p.setError(p.scanner.Err(), "scanner failed")
	return false
}

func parseRune(b string) (rune, error) {
	if len(b) > 2 && b[0] == 'U' && b[1] == '+' {
		b = b[2:]
	}
	x, err := strconv.ParseUint(b, 16, 32)
	return rune(x), err
}

func (p *Parser) parseRune(s string) rune {
	x, err := parseRune(s)
	p.setError(err, "failed to parse rune")
	return x
}

/*
// Rune parses and returns field i as a rune.
func (p *Parser) Rune(i int) rune {
    if i > 0 || p.keepRanges {
        return p.parseRune(p.getField(i))
    }
    return p.rangeStart
}

// Runes interprets and returns field i as a sequence of runes.
func (p *Parser) Runes(i int) (runes []rune) {
    add := func(s string) {
        if s = strings.TrimSpace(s); len(s) > 0 {
            runes = append(runes, p.parseRune(s))
        }
    }
    for b := p.getField(i); ; {
        i := strings.IndexByte(b, ' ')
        if i == -1 {
            add(b)
            break
        }
        add(b[:i])
        b = b[i+1:]
    }
    return
}
*/

var (
	errIncorrectLegacyRange = errors.New("ucd: unmatched <* First>")

	// reRange matches one line of a legacy rune range.
	reRange = regexp.MustCompile("^([0-9A-F]*);<([^,]*), ([^>]*)>(.*)$")
)

// Range parses and returns field i as a rune range. A range is inclusive at
// both ends. If the field only has one rune, first and last will be identical.
// It supports the legacy format for ranges used in UnicodeData.txt.
func (p *Parser) Range(i int) (first, last rune) {
	if !p.keepRanges {
		return p.rangeStart, p.rangeStart
	}
	return p.getRange(i)
}

func (p *Parser) getRange(i int) (first, last rune) {
	b := p.getField(i)
	if k := strings.Index(b, ".."); k != -1 {
		return p.parseRune(b[:k]), p.parseRune(b[k+2:])
	}
	// The first field may not be a rune, in which case we may ignore any error
	// and set the range as 0..0.
	x, err := parseRune(b)
	if err != nil {
		// Disable range parsing henceforth. This ensures that an error will be
		// returned if the user subsequently will try to parse this field as
		// a Rune.
		p.keepRanges = true
	}
	// Special case for UnicodeData that was retained for backwards compatibility.
	if i == 0 && len(p.field) > 1 && strings.HasSuffix(p.field[1], "First>") {
		if p.parsedRange {
			return p.rangeStart, p.rangeEnd
		}
		mf := reRange.FindStringSubmatch(p.scanner.Text())
		p.line++
		if mf == nil || !p.scanner.Scan() {
			p.setError(errIncorrectLegacyRange, "")
			return x, x
		}
		// Using Bytes would be more efficient here, but Text is a lot easier
		// and this is not a frequent case.
		ml := reRange.FindStringSubmatch(p.scanner.Text())
		if ml == nil || mf[2] != ml[2] || ml[3] != "Last" || mf[4] != ml[4] {
			p.setError(errIncorrectLegacyRange, "")
			return x, x
		}
		p.rangeStart, p.rangeEnd = x, p.parseRune(p.scanner.Text()[:len(ml[1])])
		p.parsedRange = true
		return p.rangeStart, p.rangeEnd
	}
	return x, x
}

// String parses and returns field i as a string value.
func (p *Parser) String(i int) string {
	return string(p.getField(i))
}

// ---------------------------------------------------------------------------

type Option func(p *Parser)

func OptionKeepRanges() func(*Parser) {
	return func(p *Parser) {
		p.keepRanges = true
	}
}
