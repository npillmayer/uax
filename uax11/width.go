package uax11

import (
	"unicode/utf8"

	jj "github.com/cloudfoundry/jibber_jabber"
	"github.com/npillmayer/uax"
	"github.com/npillmayer/uax/emoji"
	"github.com/npillmayer/uax/grapheme"
	"golang.org/x/text/language"
	"golang.org/x/text/width"
)

// Width returns the width of a grapheme, given as a byte slice, in terms of
// `en`s, where 1en stands for 1/2em, i.e. half a full width character.
// If grphm is invalid or just a zero width rune, a width of 0 is returned.
//
// If an empty context is given, LatinContext is assumed.
//
// Returns either 0, 1 (narrow character) or 2 (wide character).
//
func Width(grphm []byte, context *Context) int {
	if len(grphm) == 0 {
		return 0
	}
	start, _ := uax.PositionOfFirstLegalRune(string(grphm))
	if start != 0 { // grapheme starts with illegal code points
		//T().Debugf("start = %d, rest = %v", start, rest)
		return 0
	}
	if context == nil {
		context = makeLatinContext()
	} else if context.resolve == nil {
		context = evaluateContext(context)
	}
	return graphemeWidth(grphm, context)
}

// StringWidth calculates the width of a grapheme.String in terms of
// `en`s, where 1en stands for 1/2em, i.e. half a full width character.
//
// If an empty context is given, LatinContext is assumed.
//
//     s := grapheme.StringFromString("A (世). 😀")
//     w := uax11.StringWidth(s, uax11.LatinContext)
//     fmt.Printf("string has fixed-width display length of %d en", w)     ⇒  10
//
func StringWidth(s grapheme.String, context *Context) int {
	l := s.Len()
	if l == 0 {
		return 0
	}
	if context == nil {
		context = makeLatinContext()
	} else if context.resolve == nil {
		context = evaluateContext(context)
	}
	w := 0
	for i := 0; i < l; i++ {
		nth := []byte(s.Nth(i))
		w += graphemeWidth(nth, context)
	}
	return w
}

// width of a single grapheme in context
func graphemeWidth(grphm []byte, context *Context) int {
	r, _ := utf8.DecodeRune(grphm)
	//T().Debugf("grapheme '%v' => rune %#U", grphm, r)
	if emoji.EmojisClassForRune(r) >= 0 {
		//T().Debugf("%#U is emoji", r)
		return 2
	}
	cat1 := width.LookupRune(r).Kind()
	cat := context.resolve(grphm, cat1)
	//T().Debugf("cat(%#U) = %d  =>  %d", r, cat1, cat)
	if cat == width.EastAsianWide {
		return 2
	}
	return 1
}

// --- Context ---------------------------------------------------------------

// Context represents information about the typesetting environment.
//
// From UAX#11:
// The term context as used here includes extra information such as explicit
// markup, knowledge of the source code page, font information, or language and
// script identification
//
// Clients may fill a context paritially and hand it over to uax11. The functions
// in this package will try to derive a meaningful context from a partially filled one.
// This package relies on https://pkg.go.dev/golang.org/x/text/language/ for this
// to work.
//
//    context := &Context{Locale: "zh"}   // unspecified Chinese
//    _ = Width([]byte("世"), context)
//    fmt.Printf("%v", context.Script)    ⇒    “Hans”  (simplified Chinese script)
//
// Alternatively, clients may use one of the pre-defined contexts or use
// `ContextFromEnvironment` to get a client-machine dependent one.
//
type Context struct {
	ForceEastAsian bool            // force East Asian context
	Script         language.Script // ISO 15924 script identifier
	Locale         string          // ISO 639/3166 locale string
	resolve        resolver
}

// EastAsianContext is a context for East Asian languages.
var EastAsianContext = makeEastAsianContext()

// LatinContext is a context for western languages.
var LatinContext = makeLatinContext()

func makeEastAsianContext() *Context {
	ctx := &Context{
		ForceEastAsian: true,
		Script:         language.MustParseScript("Hant"),
		Locale:         "zh-Hant",
		resolve:        resolveToWide,
	}
	return ctx
}

func makeLatinContext() *Context {
	ctx := &Context{
		ForceEastAsian: false,
		Script:         language.MustParseScript("Latn"),
		Locale:         "en-US",
		resolve:        resolveToNarrow,
	}
	return ctx
}

// A resolver is used for resolving categories N and A to either Na or W.
type resolver func([]byte, width.Kind) width.Kind

func resolveToNarrow(grphm []byte, cat width.Kind) width.Kind {
	switch cat {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return width.EastAsianWide
	}
	return width.EastAsianNarrow
}

func resolveToWide(grphm []byte, cat width.Kind) width.Kind {
	switch cat {
	case width.Neutral, width.EastAsianAmbiguous, width.EastAsianWide, width.EastAsianFullwidth:
		return width.EastAsianWide
	}
	return width.EastAsianNarrow
}

// evaluateContext checks the 'input-fields' of a context and sets a
// resolver accordingly.
func evaluateContext(ctx *Context) *Context {
	if ctx.ForceEastAsian {
		ctx.resolve = resolveToWide
	} else {
		lang := language.Make(ctx.Locale)
		ctx = findResolver(lang, ctx)
	}
	return ctx
}

func findResolver(lang language.Tag, ctx *Context) *Context {
	scrcode := ctx.Script.String()
	if scrcode == "Zzzz" {
		ctx.Script, _ = lang.Script()
		scrcode = ctx.Script.String()
	}
	switch scrcode {
	case
		// East Asian
		"Bopo", "Hanb", "Hani", "Hans",
		"Hant", "Hang", "Hira", "Kana",
		"Lana", "Kitl", "Kits", "Nkdb",
		"Nkgb", "Plrd",
		// South East Asian
		"Batk", "Beng", "Bugi", "Mymr",
		"Cham", "Java", "Khmr", "Laoo",
		"Lisu", "Mtei", "Thai", "Yiii",
		"Bali", "Khar", "Rjng", "Roro",
		"Tglg", "Wole", "Buhd", "Tagb":
		ctx.resolve = resolveToWide
		return ctx
	}
	_, _, confidence := eaMatch.Match(lang)
	if confidence == language.No {
		ctx.resolve = resolveToNarrow
	} else {
		ctx.resolve = resolveToWide
	}
	return ctx
}

// A matcher for CJK and some other East Asian languages.
var eaMatch = language.NewMatcher([]language.Tag{
	language.Chinese, // The first language is used as fallback.
	language.Japanese,
	language.Korean,
	language.Vietnamese,
	language.Thai,
	language.Mongolian,
	language.Burmese,
	language.Khmer,
})

// ContextFromEnvironment creates a Context from the operating system environment,
// i.e. either from environment variables on *nix sytems of from a kernel call
// on Windows systems.
// (We rely on http://github.com/cloudfoundry/jibber_jabber for this).
//
func ContextFromEnvironment() *Context {
	userLocale, err := jj.DetectIETF()
	if err != nil {
		T().Errorf(err.Error())
		userLocale = "en-US"
		T().Infof("UAX#11 sets default user locale %v", userLocale)
	} else {
		T().Infof("UAX#11 detected user locale %v", userLocale)
	}
	lang := language.Make(userLocale)
	script, _ := lang.Script()
	ctx := &Context{
		Script: script,
		Locale: userLocale,
	}
	ctx = findResolver(lang, ctx)
	return ctx
}
