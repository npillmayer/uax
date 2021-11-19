/* Package ucdparse provides a parser for Unicode Character Database files.

Package ucdparse provides a parser for Unicode Character Database files, the
format of which is defined in http://www.unicode.org/reports/tr44/. See
http://www.unicode.org/Public/UCD/latest/ucd/ for example files.
*/
package ucdparse

import "fmt"

// scannerToken is a type for communicating between the line-level scanner and the parser.
// The scanner will read lines and wrap the content into parser tags, i.e., tokens for the
// parser to perform its operations on.
type scannerToken struct {
	LineNo, ColNo int              // start of the tag within the input source
	TokenType     scannerTokenType // type of token
	runeFrom      rune             // first/single rune
	runeTo        rune             // final rune of range (may be identical to runeFrom)
	Fields        []string         // UTF-8 content of the line (without indent and item tag)
	Comment       string           // rest-of-line comment of data item lines
	Error         error            // error condition, if any
}

//go:generate stringer -type=scannerTokenType
type scannerTokenType int8

const (
	undefined scannerTokenType = iota
	eof
	emptyDocument
	docRoot
	singleDataItem
	rangeDataItem
)

// newScannerToken creates a parser token initialized with line and column index.
func newScannerToken(line, col int) *scannerToken {
	return &scannerToken{
		LineNo: line,
		ColNo:  col,
		Fields: []string{},
	}
}

func (token *scannerToken) String() string {
	return fmt.Sprintf("token[at(%d,%d) %#U..%#U type=%s %#v]", token.LineNo, token.ColNo,
		token.runeFrom, token.runeTo, token.TokenType, token.Fields)
}

// Field gets field #1 (1…n) from the current data item.
func (token *scannerToken) Field(i int) string {
	if len(token.Fields) > 0 && i <= len(token.Fields) {
		return token.Fields[i-1]
	}
	return ""
}

// Range gets the character range from the current data item.
func (token *scannerToken) Range() (from, to rune) {
	return token.runeFrom, token.runeTo
}
