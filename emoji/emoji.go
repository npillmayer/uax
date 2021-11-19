/*
Package emoji implements Unicode UTS #51 emoji classes.

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>

Attention

Before using emoji classes, clients will have to initialize them.

  SetupEmojiClasses()

This initializes all the code-point range tables. Initialization is
not done beforehand, as it consumes quite some memory. */
package emoji

import (
	"sync"
	"unicode"
)

// EmojisClassForRune is the top-level client function:
// Get the emoji class for a Unicode code-point
// Will return -1 if the code-point has no emoji-class.
func EmojisClassForRune(r rune) EmojisClass {
	for c := EmojisClass(0); c <= Extended_PictographicClass; c++ {
		urange := rangeFromEmojisClass[c]
		if urange != nil && unicode.Is(urange, r) {
			return c
		}
	}
	return -1
}

var setupOnce sync.Once

// SetupEmojisClasses is the top-level preparation function:
// Create code-point classes for emojis.
// (Concurrency-safe).
func SetupEmojisClasses() {
	setupOnce.Do(setupEmojisClasses)
}
