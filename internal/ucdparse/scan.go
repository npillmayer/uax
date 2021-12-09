package ucdparse

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// --- Line level scanner ----------------------------------------------------

// scanner is a type for a line-level scanner.
//
// Our line-level scanner will operate by calling scanning steps in a chain, iteratively.
// Each step function tests for valid lookahead and then possibly branches out to a
// subsequent step function. Step functions may consume input characters ("match(…)").
//
type scanner struct {
	Buf       *lineBuffer // line buffer abstracts away properties of input readers
	Step      scannerStep // the next scanner step to execute in a chain
	LastError error       // last error, if any
	Token     *Token      // last token produced by scanner
}

// We're buiding up a scanner from chains of scanner step functions.
// Tokens may be modified by a step function.
// A scanner step will return the next step in the chain, or nil to stop/accept.
//
type scannerStep func(*Token) (*Token, scannerStep)

// New creates a scanner for an input reader.
func New(inputReader io.Reader) (*scanner, error) {
	if inputReader == nil {
		return nil, errors.New("no input present")
	}
	buf := newLineBuffer(inputReader)
	sc := &scanner{Buf: buf}
	sc.Step = sc.ScanFileStart
	return sc, nil
}

// Parse iterates over each line of the data file and calls callback f on it.
func Parse(r io.Reader, f func(token *Token)) error {
	sc, err := New(r)
	if err != nil {
		return err
	}
	for sc.Next() {
		f(sc.Token)
	}
	return sc.LastError
}

// Next is called to receive the next line-level token. A token
// subsumes the properties of a line of UCD input.
//
// Next usually will iterate over a chain of step functions until it reaches an
// accepting state. Acceptance is signalled by getting a nil-step return value from a
// step function, meaning there is no further step applicable in this chain.
//
// If a step function returns an error-signalling token, the chaining stops as well.
//
func (sc *scanner) Next() bool {
	sc.Token = newScannerToken(sc.Buf.CurrentLine, int(sc.Buf.Cursor))
	if sc.Buf.IsEof() {
		sc.Token.TokenType = eof
		return false
	}
	if sc.Step == nil {
		sc.Step = sc.ScanItem
	}
	for sc.Step != nil {
		sc.Token, sc.Step = sc.Step(sc.Token)
		if sc.Token == nil {
			sc.Buf.AdvanceLine()
			break
		} else if sc.Token.Error != nil {
			sc.LastError = sc.Token.Error
			sc.Buf.AdvanceLine()
			break
		}
		if sc.Buf.Line.Size() == 0 {
			//fmt.Printf("===> line empty\n")
			break
		}
	}
	fmt.Printf("# new %s\n", sc.Token)
	return true
}

// ScanFileStart matches a valid start of a UCD document input. This is always the
// first step function to call.
//
//    file start:
//      -> EOF:   emptyDocument
//      -> other: docRoot
//
func (sc *scanner) ScanFileStart(token *Token) (*Token, scannerStep) {
	token.TokenType = emptyDocument
	if sc.Buf == nil {
		token.Error = errors.New("no valid input document")
		return token, nil
	}
	if sc.Buf.IsEof() {
		return token, nil
	}
	token.TokenType = docRoot
	if sc.Buf.Lookahead == ' ' {
		// From the spec: There is no indentation on the top-level object.
		token.Error = errors.New("top-level item must not be indented")
	}
	return token, nil
}

// StepItem is a step function to start recognizing a line-level item.
func (sc *scanner) ScanItem(token *Token) (*Token, scannerStep) {
	fmt.Println("---> ScanItem")
	return token, sc.ScanRuneRange
}

func (sc *scanner) ScanRuneRange(token *Token) (*Token, scannerStep) {
	la := sc.Buf.Lookahead
	marker := sc.Buf.ByteCursor
	if marker > 0 {
		marker--
	}
	for isHexDigit(la) {
		sc.Buf.match(singleRune(la))
		la = sc.Buf.Lookahead
	}
	pos := sc.Buf.ByteCursor - 1
	if pos > marker {
		fmt.Printf("hex word %s\n", sc.Buf.Text[marker:pos])
		hex := sc.Buf.Text[marker:pos]
		if n, err := strconv.ParseInt(hex, 16, 32); err != nil {
			fmt.Printf("hex decoding error: %v", err)
			token.Error = fmt.Errorf("hex decoding error: %w", err)
			return token, nil
		} else {
			token.runeFrom = rune(n)
			token.runeTo = rune(n)
		}
		fmt.Printf("rune 1 = %v\n", token.runeFrom)
	}
	var isRange bool
	fmt.Printf("LA = %#U\n", la)
	for la == '.' {
		isRange = true
		sc.Buf.match(singleRune(la))
		la = sc.Buf.Lookahead
	}
	fmt.Printf("is range = %v\n", isRange)
	if !isRange {
		token.TokenType = singleDataItem
		return token, sc.ScanItemBody
	}
	marker = sc.Buf.ByteCursor - 1
	for isHexDigit(la) {
		sc.Buf.match(singleRune(la))
		la = sc.Buf.Lookahead
	}
	pos = sc.Buf.ByteCursor - 1
	if pos > marker {
		fmt.Printf("hex word %s\n", sc.Buf.Text[marker:pos])
		hex := sc.Buf.Text[marker:pos]
		if n, err := strconv.ParseInt(hex, 16, 32); err != nil {
			fmt.Printf("hex decoding error: %v", err)
			token.Error = fmt.Errorf("hex decoding error: %w", err)
			return token, nil
		} else {
			token.runeTo = rune(n)
		}
		fmt.Printf("rune 2 = %v\n", token.runeTo)
	}
	token.TokenType = rangeDataItem
	fmt.Printf("token = %v\n", token)
	return token, sc.ScanItemBody
}

func isHexDigit(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func (sc *scanner) ScanItemBody(token *Token) (*Token, scannerStep) {
	rest := sc.Buf.ReadLineRemainder()
	fmt.Printf("remainder = %q\n", rest)
	a := strings.Split(rest, "#")
	if len(a) > 1 {
		token.Comment = strings.TrimSpace(a[1])
		fmt.Printf("@ comment = %q\n", token.Comment)
	}
	a[0] = strings.TrimSpace(a[0])
	a[0] = strings.TrimLeft(a[0], ";")
	token.Fields = strings.Split(a[0], ";")
	return token, nil
}
