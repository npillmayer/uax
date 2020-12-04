/*
Package emoji implements Unicode UTS #51 emoji classes.

BSD License

Copyright (c) 2017-20, Norbert Pillmayer

All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of Norbert Pillmayer nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

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
