package testdata

import _ "embed"

//go:embed ucd/BidiBrackets.txt
var BidiBrackets []byte

//go:embed ucd/BidiCharacterTest.txt
var BidiCharacterTest []byte

//go:embed ucd/auxiliary/GraphemeBreakProperty.txt
var GraphemeBreakProperty []byte

//go:embed ucd/auxiliary/GraphemeBreakTest.txt
var GraphemeBreakTest []byte

//go:embed ucd/LineBreak.txt
var LineBreak []byte

//go:embed ucd/auxiliary/LineBreakTest.txt
var LineBreakTest []byte

//go:embed ucd/auxiliary/WordBreakProperty.txt
var WordBreakProperty []byte

//go:embed ucd/auxiliary/WordBreakTest.txt
var WordBreakTest []byte

//go:embed ucd/emoji/emoji-data.txt
var EmojiBreakProperty []byte
