// Code generated by github.com/npillmayer/uax/internal/classgen DO NOT EDIT
//
// BSD License, Copyright (c) 2018, Norbert Pillmayer (norbert@pillmayer.com)

package grapheme

import (
	"strconv"
	"unicode"
)

// Class for grapheme.
// Must be convertable to int.
type Class int

const (
	CRClass                 Class = 0
	ControlClass            Class = 1
	ExtendClass             Class = 2
	LClass                  Class = 3
	LFClass                 Class = 4
	LVClass                 Class = 5
	LVTClass                Class = 6
	PrependClass            Class = 7
	Regional_IndicatorClass Class = 8
	SpacingMarkClass        Class = 9
	TClass                  Class = 10
	VClass                  Class = 11
	ZWJClass                Class = 12

	Other Class = -1 // pseudo class for any other
	sot   Class = -2 // pseudo class "start of text"
	eot   Class = -3 // pseudo class "end of text"
)

// String returns the Class name.
func (c Class) String() string {
	switch c {
	case Other:
		return "Other"
	case sot:
		return "sot"
	case eot:
		return "eot"
	default:
		return "Class(" + strconv.Itoa(int(c)) + ")"
	case CRClass:
		return "CRClass"
	case ControlClass:
		return "ControlClass"
	case ExtendClass:
		return "ExtendClass"
	case LClass:
		return "LClass"
	case LFClass:
		return "LFClass"
	case LVClass:
		return "LVClass"
	case LVTClass:
		return "LVTClass"
	case PrependClass:
		return "PrependClass"
	case Regional_IndicatorClass:
		return "Regional_IndicatorClass"
	case SpacingMarkClass:
		return "SpacingMarkClass"
	case TClass:
		return "TClass"
	case VClass:
		return "VClass"
	case ZWJClass:
		return "ZWJClass"
	}
}

var rangeFromClass = []*unicode.RangeTable{
	CRClass:                 CR,
	ControlClass:            Control,
	ExtendClass:             Extend,
	LClass:                  L,
	LFClass:                 LF,
	LVClass:                 LV,
	LVTClass:                LVT,
	PrependClass:            Prepend,
	Regional_IndicatorClass: Regional_Indicator,
	SpacingMarkClass:        SpacingMark,
	TClass:                  T,
	VClass:                  V,
	ZWJClass:                ZWJ,
}

// Range tables for grapheme classes.
// Clients can check with unicode.Is(..., rune)
var (
	CR                 = _CR
	Control            = _Control
	Extend             = _Extend
	L                  = _L
	LF                 = _LF
	LV                 = _LV
	LVT                = _LVT
	Prepend            = _Prepend
	Regional_Indicator = _Regional_Indicator
	SpacingMark        = _SpacingMark
	T                  = _T
	V                  = _V
	ZWJ                = _ZWJ
)

// size 62 bytes (0.06 KiB)
var _CR = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0xd, 0xd, 1},
	},
	LatinOffset: 1,
}

// size 188 bytes (0.18 KiB)
var _Control = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x0, 0x9, 1},
		{0xb, 0xc, 1},
		{0xe, 0x1f, 1},
		{0x7f, 0x9f, 1},
		{0xad, 0x61c, 1391},
		{0x180e, 0x200b, 2045},
		{0x200e, 0x200f, 1},
		{0x2028, 0x202e, 1},
		{0x2060, 0x206f, 1},
		{0xd800, 0xdfff, 1},
		{0xfeff, 0xfff0, 241},
		{0xfff1, 0xfffb, 1},
	},
	R32: []unicode.Range32{
		{0x1bca0, 0x1bca3, 1},
		{0x1d173, 0x1d17a, 1},
		{0xe0000, 0xe001f, 1},
		{0xe0080, 0xe00ff, 1},
		{0xe01f0, 0xe0fff, 1},
	},
	LatinOffset: 4,
}

// size 2444 bytes (2.39 KiB)
var _Extend = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x300, 0x36f, 1},
		{0x483, 0x489, 1},
		{0x591, 0x5bd, 1},
		{0x5bf, 0x5c1, 2},
		{0x5c2, 0x5c4, 2},
		{0x5c5, 0x5c7, 2},
		{0x610, 0x61a, 1},
		{0x64b, 0x65f, 1},
		{0x670, 0x6d6, 102},
		{0x6d7, 0x6dc, 1},
		{0x6df, 0x6e4, 1},
		{0x6e7, 0x6e8, 1},
		{0x6ea, 0x6ed, 1},
		{0x711, 0x730, 31},
		{0x731, 0x74a, 1},
		{0x7a6, 0x7b0, 1},
		{0x7eb, 0x7f3, 1},
		{0x7fd, 0x816, 25},
		{0x817, 0x819, 1},
		{0x81b, 0x823, 1},
		{0x825, 0x827, 1},
		{0x829, 0x82d, 1},
		{0x859, 0x85b, 1},
		{0x8d3, 0x8e1, 1},
		{0x8e3, 0x902, 1},
		{0x93a, 0x93c, 2},
		{0x941, 0x948, 1},
		{0x94d, 0x951, 4},
		{0x952, 0x957, 1},
		{0x962, 0x963, 1},
		{0x981, 0x9bc, 59},
		{0x9be, 0x9c1, 3},
		{0x9c2, 0x9c4, 1},
		{0x9cd, 0x9d7, 10},
		{0x9e2, 0x9e3, 1},
		{0x9fe, 0xa01, 3},
		{0xa02, 0xa3c, 58},
		{0xa41, 0xa42, 1},
		{0xa47, 0xa48, 1},
		{0xa4b, 0xa4d, 1},
		{0xa51, 0xa70, 31},
		{0xa71, 0xa75, 4},
		{0xa81, 0xa82, 1},
		{0xabc, 0xac1, 5},
		{0xac2, 0xac5, 1},
		{0xac7, 0xac8, 1},
		{0xacd, 0xae2, 21},
		{0xae3, 0xafa, 23},
		{0xafb, 0xaff, 1},
		{0xb01, 0xb3c, 59},
		{0xb3e, 0xb3f, 1},
		{0xb41, 0xb44, 1},
		{0xb4d, 0xb56, 9},
		{0xb57, 0xb62, 11},
		{0xb63, 0xb82, 31},
		{0xbbe, 0xbc0, 2},
		{0xbcd, 0xbd7, 10},
		{0xc00, 0xc04, 4},
		{0xc3e, 0xc40, 1},
		{0xc46, 0xc48, 1},
		{0xc4a, 0xc4d, 1},
		{0xc55, 0xc56, 1},
		{0xc62, 0xc63, 1},
		{0xc81, 0xcbc, 59},
		{0xcbf, 0xcc2, 3},
		{0xcc6, 0xccc, 6},
		{0xccd, 0xcd5, 8},
		{0xcd6, 0xce2, 12},
		{0xce3, 0xd00, 29},
		{0xd01, 0xd3b, 58},
		{0xd3c, 0xd3e, 2},
		{0xd41, 0xd44, 1},
		{0xd4d, 0xd57, 10},
		{0xd62, 0xd63, 1},
		{0xdca, 0xdcf, 5},
		{0xdd2, 0xdd4, 1},
		{0xdd6, 0xddf, 9},
		{0xe31, 0xe34, 3},
		{0xe35, 0xe3a, 1},
		{0xe47, 0xe4e, 1},
		{0xeb1, 0xeb4, 3},
		{0xeb5, 0xeb9, 1},
		{0xebb, 0xebc, 1},
		{0xec8, 0xecd, 1},
		{0xf18, 0xf19, 1},
		{0xf35, 0xf39, 2},
		{0xf71, 0xf7e, 1},
		{0xf80, 0xf84, 1},
		{0xf86, 0xf87, 1},
		{0xf8d, 0xf97, 1},
		{0xf99, 0xfbc, 1},
		{0xfc6, 0x102d, 103},
		{0x102e, 0x1030, 1},
		{0x1032, 0x1037, 1},
		{0x1039, 0x103a, 1},
		{0x103d, 0x103e, 1},
		{0x1058, 0x1059, 1},
		{0x105e, 0x1060, 1},
		{0x1071, 0x1074, 1},
		{0x1082, 0x1085, 3},
		{0x1086, 0x108d, 7},
		{0x109d, 0x135d, 704},
		{0x135e, 0x135f, 1},
		{0x1712, 0x1714, 1},
		{0x1732, 0x1734, 1},
		{0x1752, 0x1753, 1},
		{0x1772, 0x1773, 1},
		{0x17b4, 0x17b5, 1},
		{0x17b7, 0x17bd, 1},
		{0x17c6, 0x17c9, 3},
		{0x17ca, 0x17d3, 1},
		{0x17dd, 0x180b, 46},
		{0x180c, 0x180d, 1},
		{0x1885, 0x1886, 1},
		{0x18a9, 0x1920, 119},
		{0x1921, 0x1922, 1},
		{0x1927, 0x1928, 1},
		{0x1932, 0x1939, 7},
		{0x193a, 0x193b, 1},
		{0x1a17, 0x1a18, 1},
		{0x1a1b, 0x1a56, 59},
		{0x1a58, 0x1a5e, 1},
		{0x1a60, 0x1a62, 2},
		{0x1a65, 0x1a6c, 1},
		{0x1a73, 0x1a7c, 1},
		{0x1a7f, 0x1ab0, 49},
		{0x1ab1, 0x1abe, 1},
		{0x1b00, 0x1b03, 1},
		{0x1b34, 0x1b36, 2},
		{0x1b37, 0x1b3a, 1},
		{0x1b3c, 0x1b42, 6},
		{0x1b6b, 0x1b73, 1},
		{0x1b80, 0x1b81, 1},
		{0x1ba2, 0x1ba5, 1},
		{0x1ba8, 0x1ba9, 1},
		{0x1bab, 0x1bad, 1},
		{0x1be6, 0x1be8, 2},
		{0x1be9, 0x1bed, 4},
		{0x1bef, 0x1bf1, 1},
		{0x1c2c, 0x1c33, 1},
		{0x1c36, 0x1c37, 1},
		{0x1cd0, 0x1cd2, 1},
		{0x1cd4, 0x1ce0, 1},
		{0x1ce2, 0x1ce8, 1},
		{0x1ced, 0x1cf4, 7},
		{0x1cf8, 0x1cf9, 1},
		{0x1dc0, 0x1df9, 1},
		{0x1dfb, 0x1dff, 1},
		{0x200c, 0x20d0, 196},
		{0x20d1, 0x20f0, 1},
		{0x2cef, 0x2cf1, 1},
		{0x2d7f, 0x2de0, 97},
		{0x2de1, 0x2dff, 1},
		{0x302a, 0x302f, 1},
		{0x3099, 0x309a, 1},
		{0xa66f, 0xa672, 1},
		{0xa674, 0xa67d, 1},
		{0xa69e, 0xa69f, 1},
		{0xa6f0, 0xa6f1, 1},
		{0xa802, 0xa806, 4},
		{0xa80b, 0xa825, 26},
		{0xa826, 0xa8c4, 158},
		{0xa8c5, 0xa8e0, 27},
		{0xa8e1, 0xa8f1, 1},
		{0xa8ff, 0xa926, 39},
		{0xa927, 0xa92d, 1},
		{0xa947, 0xa951, 1},
		{0xa980, 0xa982, 1},
		{0xa9b3, 0xa9b6, 3},
		{0xa9b7, 0xa9b9, 1},
		{0xa9bc, 0xa9e5, 41},
		{0xaa29, 0xaa2e, 1},
		{0xaa31, 0xaa32, 1},
		{0xaa35, 0xaa36, 1},
		{0xaa43, 0xaa4c, 9},
		{0xaa7c, 0xaab0, 52},
		{0xaab2, 0xaab4, 1},
		{0xaab7, 0xaab8, 1},
		{0xaabe, 0xaabf, 1},
		{0xaac1, 0xaaec, 43},
		{0xaaed, 0xaaf6, 9},
		{0xabe5, 0xabe8, 3},
		{0xabed, 0xfb1e, 20273},
		{0xfe00, 0xfe0f, 1},
		{0xfe20, 0xfe2f, 1},
		{0xff9e, 0xff9f, 1},
	},
	R32: []unicode.Range32{
		{0x101fd, 0x102e0, 227},
		{0x10376, 0x1037a, 1},
		{0x10a01, 0x10a03, 1},
		{0x10a05, 0x10a06, 1},
		{0x10a0c, 0x10a0f, 1},
		{0x10a38, 0x10a3a, 1},
		{0x10a3f, 0x10ae5, 166},
		{0x10ae6, 0x10d24, 574},
		{0x10d25, 0x10d27, 1},
		{0x10f46, 0x10f50, 1},
		{0x11001, 0x11038, 55},
		{0x11039, 0x11046, 1},
		{0x1107f, 0x11081, 1},
		{0x110b3, 0x110b6, 1},
		{0x110b9, 0x110ba, 1},
		{0x11100, 0x11102, 1},
		{0x11127, 0x1112b, 1},
		{0x1112d, 0x11134, 1},
		{0x11173, 0x11180, 13},
		{0x11181, 0x111b6, 53},
		{0x111b7, 0x111be, 1},
		{0x111c9, 0x111cc, 1},
		{0x1122f, 0x11231, 1},
		{0x11234, 0x11236, 2},
		{0x11237, 0x1123e, 7},
		{0x112df, 0x112e3, 4},
		{0x112e4, 0x112ea, 1},
		{0x11300, 0x11301, 1},
		{0x1133b, 0x1133c, 1},
		{0x1133e, 0x11340, 2},
		{0x11357, 0x11366, 15},
		{0x11367, 0x1136c, 1},
		{0x11370, 0x11374, 1},
		{0x11438, 0x1143f, 1},
		{0x11442, 0x11444, 1},
		{0x11446, 0x1145e, 24},
		{0x114b0, 0x114b3, 3},
		{0x114b4, 0x114b8, 1},
		{0x114ba, 0x114bd, 3},
		{0x114bf, 0x114c0, 1},
		{0x114c2, 0x114c3, 1},
		{0x115af, 0x115b2, 3},
		{0x115b3, 0x115b5, 1},
		{0x115bc, 0x115bd, 1},
		{0x115bf, 0x115c0, 1},
		{0x115dc, 0x115dd, 1},
		{0x11633, 0x1163a, 1},
		{0x1163d, 0x1163f, 2},
		{0x11640, 0x116ab, 107},
		{0x116ad, 0x116b0, 3},
		{0x116b1, 0x116b5, 1},
		{0x116b7, 0x1171d, 102},
		{0x1171e, 0x1171f, 1},
		{0x11722, 0x11725, 1},
		{0x11727, 0x1172b, 1},
		{0x1182f, 0x11837, 1},
		{0x11839, 0x1183a, 1},
		{0x11a01, 0x11a0a, 1},
		{0x11a33, 0x11a38, 1},
		{0x11a3b, 0x11a3e, 1},
		{0x11a47, 0x11a51, 10},
		{0x11a52, 0x11a56, 1},
		{0x11a59, 0x11a5b, 1},
		{0x11a8a, 0x11a96, 1},
		{0x11a98, 0x11a99, 1},
		{0x11c30, 0x11c36, 1},
		{0x11c38, 0x11c3d, 1},
		{0x11c3f, 0x11c92, 83},
		{0x11c93, 0x11ca7, 1},
		{0x11caa, 0x11cb0, 1},
		{0x11cb2, 0x11cb3, 1},
		{0x11cb5, 0x11cb6, 1},
		{0x11d31, 0x11d36, 1},
		{0x11d3a, 0x11d3c, 2},
		{0x11d3d, 0x11d3f, 2},
		{0x11d40, 0x11d45, 1},
		{0x11d47, 0x11d90, 73},
		{0x11d91, 0x11d95, 4},
		{0x11d97, 0x11ef3, 348},
		{0x11ef4, 0x16af0, 19452},
		{0x16af1, 0x16af4, 1},
		{0x16b30, 0x16b36, 1},
		{0x16f8f, 0x16f92, 1},
		{0x1bc9d, 0x1bc9e, 1},
		{0x1d165, 0x1d167, 2},
		{0x1d168, 0x1d169, 1},
		{0x1d16e, 0x1d172, 1},
		{0x1d17b, 0x1d182, 1},
		{0x1d185, 0x1d18b, 1},
		{0x1d1aa, 0x1d1ad, 1},
		{0x1d242, 0x1d244, 1},
		{0x1da00, 0x1da36, 1},
		{0x1da3b, 0x1da6c, 1},
		{0x1da75, 0x1da84, 15},
		{0x1da9b, 0x1da9f, 1},
		{0x1daa1, 0x1daaf, 1},
		{0x1e000, 0x1e006, 1},
		{0x1e008, 0x1e018, 1},
		{0x1e01b, 0x1e021, 1},
		{0x1e023, 0x1e024, 1},
		{0x1e026, 0x1e02a, 1},
		{0x1e8d0, 0x1e8d6, 1},
		{0x1e944, 0x1e94a, 1},
		{0x1f3fb, 0x1f3ff, 1},
		{0xe0020, 0xe007f, 1},
		{0xe0100, 0xe01ef, 1},
	},
}

// size 68 bytes (0.07 KiB)
var _L = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x1100, 0x115f, 1},
		{0xa960, 0xa97c, 1},
	},
}

// size 62 bytes (0.06 KiB)
var _LF = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0xa, 0xa, 1},
	},
	LatinOffset: 1,
}

// size 62 bytes (0.06 KiB)
var _LV = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0xac00, 0xd788, 28},
	},
}

// size 2450 bytes (2.39 KiB)
var _LVT = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0xac01, 0xac1b, 1},
		{0xac1d, 0xac37, 1},
		{0xac39, 0xac53, 1},
		{0xac55, 0xac6f, 1},
		{0xac71, 0xac8b, 1},
		{0xac8d, 0xaca7, 1},
		{0xaca9, 0xacc3, 1},
		{0xacc5, 0xacdf, 1},
		{0xace1, 0xacfb, 1},
		{0xacfd, 0xad17, 1},
		{0xad19, 0xad33, 1},
		{0xad35, 0xad4f, 1},
		{0xad51, 0xad6b, 1},
		{0xad6d, 0xad87, 1},
		{0xad89, 0xada3, 1},
		{0xada5, 0xadbf, 1},
		{0xadc1, 0xaddb, 1},
		{0xaddd, 0xadf7, 1},
		{0xadf9, 0xae13, 1},
		{0xae15, 0xae2f, 1},
		{0xae31, 0xae4b, 1},
		{0xae4d, 0xae67, 1},
		{0xae69, 0xae83, 1},
		{0xae85, 0xae9f, 1},
		{0xaea1, 0xaebb, 1},
		{0xaebd, 0xaed7, 1},
		{0xaed9, 0xaef3, 1},
		{0xaef5, 0xaf0f, 1},
		{0xaf11, 0xaf2b, 1},
		{0xaf2d, 0xaf47, 1},
		{0xaf49, 0xaf63, 1},
		{0xaf65, 0xaf7f, 1},
		{0xaf81, 0xaf9b, 1},
		{0xaf9d, 0xafb7, 1},
		{0xafb9, 0xafd3, 1},
		{0xafd5, 0xafef, 1},
		{0xaff1, 0xb00b, 1},
		{0xb00d, 0xb027, 1},
		{0xb029, 0xb043, 1},
		{0xb045, 0xb05f, 1},
		{0xb061, 0xb07b, 1},
		{0xb07d, 0xb097, 1},
		{0xb099, 0xb0b3, 1},
		{0xb0b5, 0xb0cf, 1},
		{0xb0d1, 0xb0eb, 1},
		{0xb0ed, 0xb107, 1},
		{0xb109, 0xb123, 1},
		{0xb125, 0xb13f, 1},
		{0xb141, 0xb15b, 1},
		{0xb15d, 0xb177, 1},
		{0xb179, 0xb193, 1},
		{0xb195, 0xb1af, 1},
		{0xb1b1, 0xb1cb, 1},
		{0xb1cd, 0xb1e7, 1},
		{0xb1e9, 0xb203, 1},
		{0xb205, 0xb21f, 1},
		{0xb221, 0xb23b, 1},
		{0xb23d, 0xb257, 1},
		{0xb259, 0xb273, 1},
		{0xb275, 0xb28f, 1},
		{0xb291, 0xb2ab, 1},
		{0xb2ad, 0xb2c7, 1},
		{0xb2c9, 0xb2e3, 1},
		{0xb2e5, 0xb2ff, 1},
		{0xb301, 0xb31b, 1},
		{0xb31d, 0xb337, 1},
		{0xb339, 0xb353, 1},
		{0xb355, 0xb36f, 1},
		{0xb371, 0xb38b, 1},
		{0xb38d, 0xb3a7, 1},
		{0xb3a9, 0xb3c3, 1},
		{0xb3c5, 0xb3df, 1},
		{0xb3e1, 0xb3fb, 1},
		{0xb3fd, 0xb417, 1},
		{0xb419, 0xb433, 1},
		{0xb435, 0xb44f, 1},
		{0xb451, 0xb46b, 1},
		{0xb46d, 0xb487, 1},
		{0xb489, 0xb4a3, 1},
		{0xb4a5, 0xb4bf, 1},
		{0xb4c1, 0xb4db, 1},
		{0xb4dd, 0xb4f7, 1},
		{0xb4f9, 0xb513, 1},
		{0xb515, 0xb52f, 1},
		{0xb531, 0xb54b, 1},
		{0xb54d, 0xb567, 1},
		{0xb569, 0xb583, 1},
		{0xb585, 0xb59f, 1},
		{0xb5a1, 0xb5bb, 1},
		{0xb5bd, 0xb5d7, 1},
		{0xb5d9, 0xb5f3, 1},
		{0xb5f5, 0xb60f, 1},
		{0xb611, 0xb62b, 1},
		{0xb62d, 0xb647, 1},
		{0xb649, 0xb663, 1},
		{0xb665, 0xb67f, 1},
		{0xb681, 0xb69b, 1},
		{0xb69d, 0xb6b7, 1},
		{0xb6b9, 0xb6d3, 1},
		{0xb6d5, 0xb6ef, 1},
		{0xb6f1, 0xb70b, 1},
		{0xb70d, 0xb727, 1},
		{0xb729, 0xb743, 1},
		{0xb745, 0xb75f, 1},
		{0xb761, 0xb77b, 1},
		{0xb77d, 0xb797, 1},
		{0xb799, 0xb7b3, 1},
		{0xb7b5, 0xb7cf, 1},
		{0xb7d1, 0xb7eb, 1},
		{0xb7ed, 0xb807, 1},
		{0xb809, 0xb823, 1},
		{0xb825, 0xb83f, 1},
		{0xb841, 0xb85b, 1},
		{0xb85d, 0xb877, 1},
		{0xb879, 0xb893, 1},
		{0xb895, 0xb8af, 1},
		{0xb8b1, 0xb8cb, 1},
		{0xb8cd, 0xb8e7, 1},
		{0xb8e9, 0xb903, 1},
		{0xb905, 0xb91f, 1},
		{0xb921, 0xb93b, 1},
		{0xb93d, 0xb957, 1},
		{0xb959, 0xb973, 1},
		{0xb975, 0xb98f, 1},
		{0xb991, 0xb9ab, 1},
		{0xb9ad, 0xb9c7, 1},
		{0xb9c9, 0xb9e3, 1},
		{0xb9e5, 0xb9ff, 1},
		{0xba01, 0xba1b, 1},
		{0xba1d, 0xba37, 1},
		{0xba39, 0xba53, 1},
		{0xba55, 0xba6f, 1},
		{0xba71, 0xba8b, 1},
		{0xba8d, 0xbaa7, 1},
		{0xbaa9, 0xbac3, 1},
		{0xbac5, 0xbadf, 1},
		{0xbae1, 0xbafb, 1},
		{0xbafd, 0xbb17, 1},
		{0xbb19, 0xbb33, 1},
		{0xbb35, 0xbb4f, 1},
		{0xbb51, 0xbb6b, 1},
		{0xbb6d, 0xbb87, 1},
		{0xbb89, 0xbba3, 1},
		{0xbba5, 0xbbbf, 1},
		{0xbbc1, 0xbbdb, 1},
		{0xbbdd, 0xbbf7, 1},
		{0xbbf9, 0xbc13, 1},
		{0xbc15, 0xbc2f, 1},
		{0xbc31, 0xbc4b, 1},
		{0xbc4d, 0xbc67, 1},
		{0xbc69, 0xbc83, 1},
		{0xbc85, 0xbc9f, 1},
		{0xbca1, 0xbcbb, 1},
		{0xbcbd, 0xbcd7, 1},
		{0xbcd9, 0xbcf3, 1},
		{0xbcf5, 0xbd0f, 1},
		{0xbd11, 0xbd2b, 1},
		{0xbd2d, 0xbd47, 1},
		{0xbd49, 0xbd63, 1},
		{0xbd65, 0xbd7f, 1},
		{0xbd81, 0xbd9b, 1},
		{0xbd9d, 0xbdb7, 1},
		{0xbdb9, 0xbdd3, 1},
		{0xbdd5, 0xbdef, 1},
		{0xbdf1, 0xbe0b, 1},
		{0xbe0d, 0xbe27, 1},
		{0xbe29, 0xbe43, 1},
		{0xbe45, 0xbe5f, 1},
		{0xbe61, 0xbe7b, 1},
		{0xbe7d, 0xbe97, 1},
		{0xbe99, 0xbeb3, 1},
		{0xbeb5, 0xbecf, 1},
		{0xbed1, 0xbeeb, 1},
		{0xbeed, 0xbf07, 1},
		{0xbf09, 0xbf23, 1},
		{0xbf25, 0xbf3f, 1},
		{0xbf41, 0xbf5b, 1},
		{0xbf5d, 0xbf77, 1},
		{0xbf79, 0xbf93, 1},
		{0xbf95, 0xbfaf, 1},
		{0xbfb1, 0xbfcb, 1},
		{0xbfcd, 0xbfe7, 1},
		{0xbfe9, 0xc003, 1},
		{0xc005, 0xc01f, 1},
		{0xc021, 0xc03b, 1},
		{0xc03d, 0xc057, 1},
		{0xc059, 0xc073, 1},
		{0xc075, 0xc08f, 1},
		{0xc091, 0xc0ab, 1},
		{0xc0ad, 0xc0c7, 1},
		{0xc0c9, 0xc0e3, 1},
		{0xc0e5, 0xc0ff, 1},
		{0xc101, 0xc11b, 1},
		{0xc11d, 0xc137, 1},
		{0xc139, 0xc153, 1},
		{0xc155, 0xc16f, 1},
		{0xc171, 0xc18b, 1},
		{0xc18d, 0xc1a7, 1},
		{0xc1a9, 0xc1c3, 1},
		{0xc1c5, 0xc1df, 1},
		{0xc1e1, 0xc1fb, 1},
		{0xc1fd, 0xc217, 1},
		{0xc219, 0xc233, 1},
		{0xc235, 0xc24f, 1},
		{0xc251, 0xc26b, 1},
		{0xc26d, 0xc287, 1},
		{0xc289, 0xc2a3, 1},
		{0xc2a5, 0xc2bf, 1},
		{0xc2c1, 0xc2db, 1},
		{0xc2dd, 0xc2f7, 1},
		{0xc2f9, 0xc313, 1},
		{0xc315, 0xc32f, 1},
		{0xc331, 0xc34b, 1},
		{0xc34d, 0xc367, 1},
		{0xc369, 0xc383, 1},
		{0xc385, 0xc39f, 1},
		{0xc3a1, 0xc3bb, 1},
		{0xc3bd, 0xc3d7, 1},
		{0xc3d9, 0xc3f3, 1},
		{0xc3f5, 0xc40f, 1},
		{0xc411, 0xc42b, 1},
		{0xc42d, 0xc447, 1},
		{0xc449, 0xc463, 1},
		{0xc465, 0xc47f, 1},
		{0xc481, 0xc49b, 1},
		{0xc49d, 0xc4b7, 1},
		{0xc4b9, 0xc4d3, 1},
		{0xc4d5, 0xc4ef, 1},
		{0xc4f1, 0xc50b, 1},
		{0xc50d, 0xc527, 1},
		{0xc529, 0xc543, 1},
		{0xc545, 0xc55f, 1},
		{0xc561, 0xc57b, 1},
		{0xc57d, 0xc597, 1},
		{0xc599, 0xc5b3, 1},
		{0xc5b5, 0xc5cf, 1},
		{0xc5d1, 0xc5eb, 1},
		{0xc5ed, 0xc607, 1},
		{0xc609, 0xc623, 1},
		{0xc625, 0xc63f, 1},
		{0xc641, 0xc65b, 1},
		{0xc65d, 0xc677, 1},
		{0xc679, 0xc693, 1},
		{0xc695, 0xc6af, 1},
		{0xc6b1, 0xc6cb, 1},
		{0xc6cd, 0xc6e7, 1},
		{0xc6e9, 0xc703, 1},
		{0xc705, 0xc71f, 1},
		{0xc721, 0xc73b, 1},
		{0xc73d, 0xc757, 1},
		{0xc759, 0xc773, 1},
		{0xc775, 0xc78f, 1},
		{0xc791, 0xc7ab, 1},
		{0xc7ad, 0xc7c7, 1},
		{0xc7c9, 0xc7e3, 1},
		{0xc7e5, 0xc7ff, 1},
		{0xc801, 0xc81b, 1},
		{0xc81d, 0xc837, 1},
		{0xc839, 0xc853, 1},
		{0xc855, 0xc86f, 1},
		{0xc871, 0xc88b, 1},
		{0xc88d, 0xc8a7, 1},
		{0xc8a9, 0xc8c3, 1},
		{0xc8c5, 0xc8df, 1},
		{0xc8e1, 0xc8fb, 1},
		{0xc8fd, 0xc917, 1},
		{0xc919, 0xc933, 1},
		{0xc935, 0xc94f, 1},
		{0xc951, 0xc96b, 1},
		{0xc96d, 0xc987, 1},
		{0xc989, 0xc9a3, 1},
		{0xc9a5, 0xc9bf, 1},
		{0xc9c1, 0xc9db, 1},
		{0xc9dd, 0xc9f7, 1},
		{0xc9f9, 0xca13, 1},
		{0xca15, 0xca2f, 1},
		{0xca31, 0xca4b, 1},
		{0xca4d, 0xca67, 1},
		{0xca69, 0xca83, 1},
		{0xca85, 0xca9f, 1},
		{0xcaa1, 0xcabb, 1},
		{0xcabd, 0xcad7, 1},
		{0xcad9, 0xcaf3, 1},
		{0xcaf5, 0xcb0f, 1},
		{0xcb11, 0xcb2b, 1},
		{0xcb2d, 0xcb47, 1},
		{0xcb49, 0xcb63, 1},
		{0xcb65, 0xcb7f, 1},
		{0xcb81, 0xcb9b, 1},
		{0xcb9d, 0xcbb7, 1},
		{0xcbb9, 0xcbd3, 1},
		{0xcbd5, 0xcbef, 1},
		{0xcbf1, 0xcc0b, 1},
		{0xcc0d, 0xcc27, 1},
		{0xcc29, 0xcc43, 1},
		{0xcc45, 0xcc5f, 1},
		{0xcc61, 0xcc7b, 1},
		{0xcc7d, 0xcc97, 1},
		{0xcc99, 0xccb3, 1},
		{0xccb5, 0xcccf, 1},
		{0xccd1, 0xcceb, 1},
		{0xcced, 0xcd07, 1},
		{0xcd09, 0xcd23, 1},
		{0xcd25, 0xcd3f, 1},
		{0xcd41, 0xcd5b, 1},
		{0xcd5d, 0xcd77, 1},
		{0xcd79, 0xcd93, 1},
		{0xcd95, 0xcdaf, 1},
		{0xcdb1, 0xcdcb, 1},
		{0xcdcd, 0xcde7, 1},
		{0xcde9, 0xce03, 1},
		{0xce05, 0xce1f, 1},
		{0xce21, 0xce3b, 1},
		{0xce3d, 0xce57, 1},
		{0xce59, 0xce73, 1},
		{0xce75, 0xce8f, 1},
		{0xce91, 0xceab, 1},
		{0xcead, 0xcec7, 1},
		{0xcec9, 0xcee3, 1},
		{0xcee5, 0xceff, 1},
		{0xcf01, 0xcf1b, 1},
		{0xcf1d, 0xcf37, 1},
		{0xcf39, 0xcf53, 1},
		{0xcf55, 0xcf6f, 1},
		{0xcf71, 0xcf8b, 1},
		{0xcf8d, 0xcfa7, 1},
		{0xcfa9, 0xcfc3, 1},
		{0xcfc5, 0xcfdf, 1},
		{0xcfe1, 0xcffb, 1},
		{0xcffd, 0xd017, 1},
		{0xd019, 0xd033, 1},
		{0xd035, 0xd04f, 1},
		{0xd051, 0xd06b, 1},
		{0xd06d, 0xd087, 1},
		{0xd089, 0xd0a3, 1},
		{0xd0a5, 0xd0bf, 1},
		{0xd0c1, 0xd0db, 1},
		{0xd0dd, 0xd0f7, 1},
		{0xd0f9, 0xd113, 1},
		{0xd115, 0xd12f, 1},
		{0xd131, 0xd14b, 1},
		{0xd14d, 0xd167, 1},
		{0xd169, 0xd183, 1},
		{0xd185, 0xd19f, 1},
		{0xd1a1, 0xd1bb, 1},
		{0xd1bd, 0xd1d7, 1},
		{0xd1d9, 0xd1f3, 1},
		{0xd1f5, 0xd20f, 1},
		{0xd211, 0xd22b, 1},
		{0xd22d, 0xd247, 1},
		{0xd249, 0xd263, 1},
		{0xd265, 0xd27f, 1},
		{0xd281, 0xd29b, 1},
		{0xd29d, 0xd2b7, 1},
		{0xd2b9, 0xd2d3, 1},
		{0xd2d5, 0xd2ef, 1},
		{0xd2f1, 0xd30b, 1},
		{0xd30d, 0xd327, 1},
		{0xd329, 0xd343, 1},
		{0xd345, 0xd35f, 1},
		{0xd361, 0xd37b, 1},
		{0xd37d, 0xd397, 1},
		{0xd399, 0xd3b3, 1},
		{0xd3b5, 0xd3cf, 1},
		{0xd3d1, 0xd3eb, 1},
		{0xd3ed, 0xd407, 1},
		{0xd409, 0xd423, 1},
		{0xd425, 0xd43f, 1},
		{0xd441, 0xd45b, 1},
		{0xd45d, 0xd477, 1},
		{0xd479, 0xd493, 1},
		{0xd495, 0xd4af, 1},
		{0xd4b1, 0xd4cb, 1},
		{0xd4cd, 0xd4e7, 1},
		{0xd4e9, 0xd503, 1},
		{0xd505, 0xd51f, 1},
		{0xd521, 0xd53b, 1},
		{0xd53d, 0xd557, 1},
		{0xd559, 0xd573, 1},
		{0xd575, 0xd58f, 1},
		{0xd591, 0xd5ab, 1},
		{0xd5ad, 0xd5c7, 1},
		{0xd5c9, 0xd5e3, 1},
		{0xd5e5, 0xd5ff, 1},
		{0xd601, 0xd61b, 1},
		{0xd61d, 0xd637, 1},
		{0xd639, 0xd653, 1},
		{0xd655, 0xd66f, 1},
		{0xd671, 0xd68b, 1},
		{0xd68d, 0xd6a7, 1},
		{0xd6a9, 0xd6c3, 1},
		{0xd6c5, 0xd6df, 1},
		{0xd6e1, 0xd6fb, 1},
		{0xd6fd, 0xd717, 1},
		{0xd719, 0xd733, 1},
		{0xd735, 0xd74f, 1},
		{0xd751, 0xd76b, 1},
		{0xd76d, 0xd787, 1},
		{0xd789, 0xd7a3, 1},
	},
}

// size 134 bytes (0.13 KiB)
var _Prepend = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x600, 0x605, 1},
		{0x6dd, 0x70f, 50},
		{0x8e2, 0xd4e, 1132},
	},
	R32: []unicode.Range32{
		{0x110bd, 0x110cd, 16},
		{0x111c2, 0x111c3, 1},
		{0x11a3a, 0x11a86, 76},
		{0x11a87, 0x11a89, 1},
		{0x11d46, 0x11d46, 1},
	},
}

// size 68 bytes (0.07 KiB)
var _Regional_Indicator = &unicode.RangeTable{
	R32: []unicode.Range32{
		{0x1f1e6, 0x1f1ff, 1},
	},
}

// size 1094 bytes (1.07 KiB)
var _SpacingMark = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x903, 0x93b, 56},
		{0x93e, 0x940, 1},
		{0x949, 0x94c, 1},
		{0x94e, 0x94f, 1},
		{0x982, 0x983, 1},
		{0x9bf, 0x9c0, 1},
		{0x9c7, 0x9c8, 1},
		{0x9cb, 0x9cc, 1},
		{0xa03, 0xa3e, 59},
		{0xa3f, 0xa40, 1},
		{0xa83, 0xabe, 59},
		{0xabf, 0xac0, 1},
		{0xac9, 0xacb, 2},
		{0xacc, 0xb02, 54},
		{0xb03, 0xb40, 61},
		{0xb47, 0xb48, 1},
		{0xb4b, 0xb4c, 1},
		{0xbbf, 0xbc1, 2},
		{0xbc2, 0xbc6, 4},
		{0xbc7, 0xbc8, 1},
		{0xbca, 0xbcc, 1},
		{0xc01, 0xc03, 1},
		{0xc41, 0xc44, 1},
		{0xc82, 0xc83, 1},
		{0xcbe, 0xcc0, 2},
		{0xcc1, 0xcc3, 2},
		{0xcc4, 0xcc7, 3},
		{0xcc8, 0xcca, 2},
		{0xccb, 0xd02, 55},
		{0xd03, 0xd3f, 60},
		{0xd40, 0xd46, 6},
		{0xd47, 0xd48, 1},
		{0xd4a, 0xd4c, 1},
		{0xd82, 0xd83, 1},
		{0xdd0, 0xdd1, 1},
		{0xdd8, 0xdde, 1},
		{0xdf2, 0xdf3, 1},
		{0xe33, 0xeb3, 128},
		{0xf3e, 0xf3f, 1},
		{0xf7f, 0x1031, 178},
		{0x103b, 0x103c, 1},
		{0x1056, 0x1057, 1},
		{0x1084, 0x17b6, 1842},
		{0x17be, 0x17c5, 1},
		{0x17c7, 0x17c8, 1},
		{0x1923, 0x1926, 1},
		{0x1929, 0x192b, 1},
		{0x1930, 0x1931, 1},
		{0x1933, 0x1938, 1},
		{0x1a19, 0x1a1a, 1},
		{0x1a55, 0x1a57, 2},
		{0x1a6d, 0x1a72, 1},
		{0x1b04, 0x1b35, 49},
		{0x1b3b, 0x1b3d, 2},
		{0x1b3e, 0x1b41, 1},
		{0x1b43, 0x1b44, 1},
		{0x1b82, 0x1ba1, 31},
		{0x1ba6, 0x1ba7, 1},
		{0x1baa, 0x1be7, 61},
		{0x1bea, 0x1bec, 1},
		{0x1bee, 0x1bf2, 4},
		{0x1bf3, 0x1c24, 49},
		{0x1c25, 0x1c2b, 1},
		{0x1c34, 0x1c35, 1},
		{0x1ce1, 0x1cf2, 17},
		{0x1cf3, 0x1cf7, 4},
		{0xa823, 0xa824, 1},
		{0xa827, 0xa880, 89},
		{0xa881, 0xa8b4, 51},
		{0xa8b5, 0xa8c3, 1},
		{0xa952, 0xa953, 1},
		{0xa983, 0xa9b4, 49},
		{0xa9b5, 0xa9ba, 5},
		{0xa9bb, 0xa9bd, 2},
		{0xa9be, 0xa9c0, 1},
		{0xaa2f, 0xaa30, 1},
		{0xaa33, 0xaa34, 1},
		{0xaa4d, 0xaaeb, 158},
		{0xaaee, 0xaaef, 1},
		{0xaaf5, 0xabe3, 238},
		{0xabe4, 0xabe6, 2},
		{0xabe7, 0xabe9, 2},
		{0xabea, 0xabec, 2},
	},
	R32: []unicode.Range32{
		{0x11000, 0x11002, 2},
		{0x11082, 0x110b0, 46},
		{0x110b1, 0x110b2, 1},
		{0x110b7, 0x110b8, 1},
		{0x1112c, 0x11145, 25},
		{0x11146, 0x11182, 60},
		{0x111b3, 0x111b5, 1},
		{0x111bf, 0x111c0, 1},
		{0x1122c, 0x1122e, 1},
		{0x11232, 0x11233, 1},
		{0x11235, 0x112e0, 171},
		{0x112e1, 0x112e2, 1},
		{0x11302, 0x11303, 1},
		{0x1133f, 0x11341, 2},
		{0x11342, 0x11344, 1},
		{0x11347, 0x11348, 1},
		{0x1134b, 0x1134d, 1},
		{0x11362, 0x11363, 1},
		{0x11435, 0x11437, 1},
		{0x11440, 0x11441, 1},
		{0x11445, 0x114b1, 108},
		{0x114b2, 0x114b9, 7},
		{0x114bb, 0x114bc, 1},
		{0x114be, 0x114c1, 3},
		{0x115b0, 0x115b1, 1},
		{0x115b8, 0x115bb, 1},
		{0x115be, 0x11630, 114},
		{0x11631, 0x11632, 1},
		{0x1163b, 0x1163c, 1},
		{0x1163e, 0x116ac, 110},
		{0x116ae, 0x116af, 1},
		{0x116b6, 0x11720, 106},
		{0x11721, 0x11726, 5},
		{0x1182c, 0x1182e, 1},
		{0x11838, 0x11a39, 513},
		{0x11a57, 0x11a58, 1},
		{0x11a97, 0x11c2f, 408},
		{0x11c3e, 0x11ca9, 107},
		{0x11cb1, 0x11cb4, 3},
		{0x11d8a, 0x11d8e, 1},
		{0x11d93, 0x11d94, 1},
		{0x11d96, 0x11ef5, 351},
		{0x11ef6, 0x16f51, 20571},
		{0x16f52, 0x16f7e, 1},
		{0x1d166, 0x1d16d, 7},
	},
}

// size 68 bytes (0.07 KiB)
var _T = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x11a8, 0x11ff, 1},
		{0xd7cb, 0xd7fb, 1},
	},
}

// size 68 bytes (0.07 KiB)
var _V = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x1160, 0x11a7, 1},
		{0xd7b0, 0xd7c6, 1},
	},
}

// size 62 bytes (0.06 KiB)
var _ZWJ = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x200d, 0x200d, 1},
	},
}
