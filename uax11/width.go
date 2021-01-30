package uax11

// East_Asian_Width properties
const (
	N  uint16 = iota // Neutral (Not East Asian)
	A                // East Asian Ambiguous
	W                // East Asian Wide
	Na               // East Asian Narrow
	H                // East Asian Halfwidth
	F                // East Asian Fullwidth
)

// Width returns the width of a single rune as proposed by the UAX#11 standard.
// Please not that this is most certainly not what clients will want to use in
// full-grown international applications. It is nevertheless provided as a low
// level API function corresponding to UAX#11.
func Width(r rune) int {
	return 0
}
