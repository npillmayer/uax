// Code generated by github.com/npillmayer/uax/internal/classgen DO NOT EDIT
//
// BSD License, Copyright (c) 2018, Norbert Pillmayer (norbert@pillmayer.com)

package shaping

import (
	"unicode"
)

// Range tables for shaping classes.
// Clients can check with unicode.Is(..., rune)
var (
	ARAB_C = _ARAB_C
	ARAB_D = _ARAB_D
	ARAB_L = _ARAB_L
	ARAB_R = _ARAB_R
	ARAB_T = _ARAB_T
	ARAB_U = _ARAB_U
)

// size 68 bytes (0.07 KiB)
var _ARAB_C = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x640, 0x7fa, 442},
		{0x180a, 0x200d, 2051},
	},
}

// size 500 bytes (0.49 KiB)
var _ARAB_D = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x620, 0x626, 6},
		{0x628, 0x62a, 2},
		{0x62b, 0x62e, 1},
		{0x633, 0x63f, 1},
		{0x641, 0x647, 1},
		{0x649, 0x64a, 1},
		{0x66e, 0x66f, 1},
		{0x678, 0x687, 1},
		{0x69a, 0x6bf, 1},
		{0x6c1, 0x6c2, 1},
		{0x6cc, 0x6d0, 2},
		{0x6d1, 0x6fa, 41},
		{0x6fb, 0x6fc, 1},
		{0x6ff, 0x712, 19},
		{0x713, 0x714, 1},
		{0x71a, 0x71d, 1},
		{0x71f, 0x727, 1},
		{0x729, 0x72d, 2},
		{0x72e, 0x74e, 32},
		{0x74f, 0x758, 1},
		{0x75c, 0x76a, 1},
		{0x76d, 0x770, 1},
		{0x772, 0x775, 3},
		{0x776, 0x777, 1},
		{0x77a, 0x77f, 1},
		{0x7ca, 0x7ea, 1},
		{0x841, 0x845, 1},
		{0x848, 0x84a, 2},
		{0x84b, 0x853, 1},
		{0x855, 0x860, 11},
		{0x862, 0x865, 1},
		{0x868, 0x8a0, 56},
		{0x8a1, 0x8a9, 1},
		{0x8af, 0x8b0, 1},
		{0x8b3, 0x8b4, 1},
		{0x8b6, 0x8b8, 1},
		{0x8ba, 0x8bd, 1},
		{0x1807, 0x1820, 25},
		{0x1821, 0x1878, 1},
		{0x1887, 0x18a8, 1},
		{0x18aa, 0xa840, 36758},
		{0xa841, 0xa871, 1},
	},
	R32: []unicode.Range32{
		{0x10ac0, 0x10ac4, 1},
		{0x10ad3, 0x10ad6, 1},
		{0x10ad8, 0x10adc, 1},
		{0x10ade, 0x10ae0, 1},
		{0x10aeb, 0x10aee, 1},
		{0x10b80, 0x10b82, 2},
		{0x10b86, 0x10b88, 1},
		{0x10b8a, 0x10b8b, 1},
		{0x10b8d, 0x10b90, 3},
		{0x10bad, 0x10bae, 1},
		{0x10d01, 0x10d21, 1},
		{0x10d23, 0x10f30, 525},
		{0x10f31, 0x10f32, 1},
		{0x10f34, 0x10f44, 1},
		{0x10f51, 0x10f53, 1},
		{0x1e900, 0x1e943, 1},
	},
}

// size 86 bytes (0.08 KiB)
var _ARAB_L = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0xa872, 0xa872, 1},
	},
	R32: []unicode.Range32{
		{0x10acd, 0x10ad7, 10},
		{0x10d00, 0x10d00, 1},
	},
}

// size 386 bytes (0.38 KiB)
var _ARAB_R = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x622, 0x625, 1},
		{0x627, 0x629, 2},
		{0x62f, 0x632, 1},
		{0x648, 0x671, 41},
		{0x672, 0x673, 1},
		{0x675, 0x677, 1},
		{0x688, 0x699, 1},
		{0x6c0, 0x6c3, 3},
		{0x6c4, 0x6cb, 1},
		{0x6cd, 0x6cf, 2},
		{0x6d2, 0x6d3, 1},
		{0x6d5, 0x6ee, 25},
		{0x6ef, 0x710, 33},
		{0x715, 0x719, 1},
		{0x71e, 0x728, 10},
		{0x72a, 0x72c, 2},
		{0x72f, 0x74d, 30},
		{0x759, 0x75b, 1},
		{0x76b, 0x76c, 1},
		{0x771, 0x773, 2},
		{0x774, 0x778, 4},
		{0x779, 0x840, 199},
		{0x846, 0x847, 1},
		{0x849, 0x854, 11},
		{0x867, 0x869, 2},
		{0x86a, 0x8aa, 64},
		{0x8ab, 0x8ac, 1},
		{0x8ae, 0x8b1, 3},
		{0x8b2, 0x8b9, 7},
	},
	R32: []unicode.Range32{
		{0x10ac5, 0x10ac9, 2},
		{0x10aca, 0x10ace, 4},
		{0x10acf, 0x10ad2, 1},
		{0x10add, 0x10ae1, 4},
		{0x10ae4, 0x10aef, 11},
		{0x10b81, 0x10b83, 2},
		{0x10b84, 0x10b85, 1},
		{0x10b89, 0x10b8c, 3},
		{0x10b8e, 0x10b8f, 1},
		{0x10b91, 0x10ba9, 24},
		{0x10baa, 0x10bac, 1},
		{0x10d22, 0x10f33, 529},
		{0x10f54, 0x10f54, 1},
	},
}

// size 68 bytes (0.07 KiB)
var _ARAB_T = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x70f, 0x1885, 4470},
		{0x1886, 0x1886, 1},
	},
}

// size 188 bytes (0.18 KiB)
var _ARAB_U = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x600, 0x605, 1},
		{0x608, 0x60b, 3},
		{0x621, 0x674, 83},
		{0x6dd, 0x856, 377},
		{0x857, 0x858, 1},
		{0x861, 0x866, 5},
		{0x8ad, 0x8e2, 53},
		{0x1806, 0x180e, 8},
		{0x1880, 0x1884, 1},
		{0x200c, 0x202f, 35},
		{0x2066, 0x2069, 1},
		{0xa873, 0xa873, 1},
	},
	R32: []unicode.Range32{
		{0x10ac6, 0x10ac8, 2},
		{0x10acb, 0x10acc, 1},
		{0x10ae2, 0x10ae3, 1},
		{0x10baf, 0x10f45, 918},
		{0x110bd, 0x110cd, 16},
	},
}
