// Code generated by "stringer -type=TokenType"; DO NOT EDIT.

package ucdparse

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[undefined-0]
	_ = x[eof-1]
	_ = x[emptyDocument-2]
	_ = x[docRoot-3]
	_ = x[singleDataItem-4]
	_ = x[rangeDataItem-5]
}

const _TokenType_name = "undefinedeofemptyDocumentdocRootsingleDataItemrangeDataItem"

var _TokenType_index = [...]uint8{0, 9, 12, 25, 32, 46, 59}

func (i TokenType) String() string {
	if i < 0 || i >= TokenType(len(_TokenType_index)-1) {
		return "TokenType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TokenType_name[_TokenType_index[i]:_TokenType_index[i+1]]
}