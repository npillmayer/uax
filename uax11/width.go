package uax11

import "unicode"

// Category is one of 6 char categories as defined in UAX#11.
type Category int8

// East_Asian_Width properties
const (
	N  Category = iota // Neutral (Not East Asian)
	A                  // East Asian Ambiguous
	W                  // East Asian Wide
	Na                 // East Asian Narrow
	H                  // East Asian Halfwidth
	F                  // East Asian Fullwidth
)

// RangeTables is an array of six Unicode range tables, for each of N, A, Na, W, H, F.
var RangeTables = [...]*unicode.RangeTable{
	_EAW_N, _EAW_A, _EAW_W, _EAW_Na, _EAW_H, _EAW_F,
}

// WidthCategory returns the width category of a single rune as proposed by the UAX#11
// standard. Please note that this is most probably not what clients will want to use in
// full-grown international applications, as it is preferable to work on graphemes
// rather than on runes. This function is nevertheless provided as a low
// level API function corresponding to UAX#11.
//
// Returns one of N, A, Na, W, H, F.
func WidthCategory(r rune) Category {
	cat := consultEAWTables(r)
	return cat
}

// UAX#11:
//  - The unassigned code points in the following blocks default to "W":
//         CJK Unified Ideographs Extension A: U+3400..U+4DBF
//         CJK Unified Ideographs:             U+4E00..U+9FFF
//         CJK Compatibility Ideographs:       U+F900..U+FAFF
//  - All undesignated code points in Planes 2 and 3, whether inside or
//      outside of allocated blocks, default to "W":
//         Plane 2:                            U+20000..U+2FFFD
//         Plane 3:                            U+30000..U+3FFFD
var _CJK_Default_W = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x3400, 0x4dbf, 1},
		{0x4e00, 0x9fff, 1},
		{0xf900, 0xfaff, 1},
	},
	R32: []unicode.Range32{
		{0x20000, 0x2fffd, 1},
		{0x30000, 0x3fffd, 1},
	},
}

func consultEAWTables(r rune) Category {
	for cat, table := range RangeTables {
		if unicode.Is(table, r) {
			return Category(cat)
		}
	}
	if unicode.Is(_CJK_Default_W, r) {
		return W
	}
	// UAX#11:
	//  - All code points, assigned or unassigned, that are not listed
	//      explicitly are given the value "N".
	return N
}
