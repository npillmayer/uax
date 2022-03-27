/*
Package emoji implements Unicode UTS #51 emoji classes.

License

This project is provided under the terms of the UNLICENSE or
the 3-Clause BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: 'Unlicense' OR 'BSD-3-Clause'

You may use the project under the terms of either license.

Licenses are reproduced in the license file in the root folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>

Attention

Before using emoji classes, clients will have to initialize them.

  SetupEmojiClasses()

This initializes all the code-point range tables. Initialization is
not done beforehand, as it consumes quite some memory. */
package emoji

import (
	"unicode"
)

//go:generate go run ./internal/gen

// ClassForRune is the top-level client function:
// Get the emoji class for a Unicode code-point
// Will return -1 if the code-point has no emoji-class.
func ClassForRune(r rune) Class {
	for class, rt := range rangeFromClass {
		if unicode.Is(rt, r) {
			return Class(class)
		}
	}
	return Other
}
