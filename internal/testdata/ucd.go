package testdata

import _ "embed"

//go:embed ucd/BidiBrackets.txt
var BidiBrackets []byte

//go:embed ucd/BidiCharacterTest.txt
var BidiCharacterTest []byte

//go:embed ucd/auxiliary/GraphemeBreakTest.txt
var GraphemeBreakTest []byte

//go:embed ucd/auxiliary/LineBreakTest.txt
var LineBreakTest []byte

//go:embed ucd/auxiliary/WordBreakTest.txt
var WordBreakTest []byte
