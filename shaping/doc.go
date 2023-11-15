/*
Package shaping provides tables corresponding to Unicode® Character Data tables relevant
for text shaping.

___________________________________________________________________________

License

This project is provided under the terms of the UNLICENSE or
the 3-Clause BSD license denoted by the following SPDX identifier:

SPDX-License-Identifier: 'Unlicense' OR 'BSD-3-Clause'

You may use the project under the terms of either license.

Licenses are reproduced in the license file in the root folder of this module.

Copyright © 2021 Norbert Pillmayer <norbert@pillmayer.com>


*/
package shaping

//go:generate go run ../internal/classgen -f 3 -o arabictables.go -x ARAB_ -u ArabicShaping.txt -noclass
//go:generate go run ../internal/classgen -f 2 -o uipctables.go -x UIPC_ -u IndicPositionalCategory.txt -noclass
//go:generate go run ../internal/classgen -f 2 -o uisctables.go -x UISC_ -u IndicSyllabicCategory.txt -noclass
