// Code generated by "stringer -type=scannerTokenType"; DO NOT EDIT.

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

const _scannerTokenType_name = "undefinedeofemptyDocumentdocRootsingleDataItemrangeDataItem"

var _scannerTokenType_index = [...]uint8{0, 9, 12, 25, 32, 46, 59}

func (i scannerTokenType) String() string {
	if i < 0 || i >= scannerTokenType(len(_scannerTokenType_index)-1) {
		return "scannerTokenType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _scannerTokenType_name[_scannerTokenType_index[i]:_scannerTokenType_index[i+1]]
}
