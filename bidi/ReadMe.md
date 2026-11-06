## Unicode Bidirectional Algorithm

Package bidi will implement a variant of the
[Unicode UAX#9 Bidirectional Algorithm](http://www.unicode.org/reports/tr9).
It is not fully standards-conforming, but good enough for practical purposes.

Unicode Annex UAX#9 presents an algorithm to identify directional runs within
texts. The algorithm deals with characters and character runs, which UAX#9
maps to Bidi character classes. Bidi classes are then grouped according to
certain rules to determine writing directions. The algorithm is not perfect and
there are some cases where manual overriding will be necessary to produce correct
output, but it is good enough for many real-life cases.

### Deviations from the Standard

This package will interpret some of the Bidi algorithm's rules a bit differently
than a strict adhering to the standard would require, the reason being that we
postulate some general requirements which make it hard to conform to the standard
100%. The main general requirement is a restriction of the mode of access for the
input text: We operate on an `io.Reader` and do not buffer the characters read from
it. As a consequence, we will never travel backwards over characters and will never
read a character twice. However, some parts of the UAX#9 algorithm are presented
as operations on “look-behinds,” or as setting properties per character (Bidi class,
embedding level) or as a multi-pass approach. This package employs strategies borrowed
from parsing theory to arrive at the same results as the original UAX#9 algorithm.

That said, this package will implement UAX#9 in a way that conforms to the standard
for “reasonable texts”, i.e. text produced by humans for humans. Deviation from the
standard is confined to areas of the standard that deal with rather obscure border
cases. As an example, the Bidi Annex postulates a clear maximum nesting level of
bracket pairings (63 levels) per isolating run sequence. However, this package
will ignore this boundary in a certain case when markers ending an isolating run
sequence go missing. The only clients to ever recognize this deviation are most
probably UAX#9 conformity tests.

There is one limitation, however, which ignores the standard in a relevant way:
We do not implement legacy formatting directives, which the Annex calls
“Explicit Directional Embedding and Override Formatting Characters”, i.e. the
formatting directives LRE, RLE, LRO, RLO and PDF. Unicode recommends sticking
to the more modern “Isolate Formatting Characters” LRI, RLI, FSI and PDI.
This package will deal with isolate run sequences produced by isolate formatting
characters (or external markup) only. The need to deal with legacy formatting
characters may arise in the future, but currently I do not plan to implement them.

### API

As the algorithms in this package will not copy any input characters, it leaves
the burden to store the text to the calling client. This package will return
Bidi runs as intervals of text positions, which means clients must be able to
reproduce the text identified by text position. That's trivially true for text
stored in a bytes buffer or string, but one can imagine other situations where
this requirement involves some additional effort, like an input stream read from
a file.

Attention: Work in progress, not yet fully functional.

### Using the package from a client

Import `github.com/npillmayer/uax/bidi`. The module currently requires Go 1.25.
The client owns the original text and supplies one paragraph at a time through
an `io.Reader`. Keep that text available: the package returns positions, not
copies of the text. All public text positions are **UTF-8 byte offsets**, with
intervals expressed as `[from, to)`; they are not rune or grapheme indices.

The intended processing sequence is:

1. Call `ResolveParagraph(reader, markup, options...)` to obtain
   `*ResolvedLevels`. Pass `nil` for `markup` when there are no external isolate
   boundaries. The default paragraph direction is `LeftToRight`; use
   `DefaultDirection(RightToLeft)` when the surrounding layout requires RTL.
   This setting is explicit, rather than automatic first-strong detection.
2. Range over `levels.LogicalSegments()` to get source-order byte intervals and
   their resolved directions, including text inside nested isolates. Intersect
   these intervals with the client's font/style spans, then shape and measure
   the resulting items. This does not change `levels`; do not call `Split` at
   style boundaries.
3. Have the client's layout code choose line breaks from the measured items.
   When a paragraph occupies
   several lines, use `levels.Split(at, false)` to obtain the prefix and suffix
   at a byte boundary, then continue splitting the suffix. With `false`, offsets
   remain relative to the original paragraph. Choose valid text boundaries;
   `Split` does not find break opportunities or check grapheme boundaries.
4. Call `Reorder()` on each line's resolved levels. If the paragraph is a single
   line, call it directly on the result of `ResolveParagraph`. The returned
   `*Ordering` exposes `Runs`, each with a `Dir` and a `Length` in bytes. A run
   can contain several noncontiguous source intervals, so use its segment
   iterator to retrieve text rather than deriving offsets from `Length`.
5. For each run, range over `run.Segments(reverse)`. Each `Segment` contains
   `Dir`, `From`, and `To`. The iterator only orders intervals; it does not
   reverse the characters inside `text[segment.From:segment.To]`.
   For the left-to-right output convention illustrated below, the documented
   choice is `run.IsOpposite(bidi.LeftToRight)` for `reverse`.

The client remains responsible for shaping, glyph placement, mirroring, and
line-boundary whitespace handling (UAX#9 rule L1). Pass each segment's logical
text and direction to the appropriate shaping/rendering stage. A renderer that
already performs paragraph bidi layout needs a different integration to avoid
applying reordering twice. For a simple visual-text demonstration, reversing
RTL text would need to preserve grapheme clusters; reversing UTF-8 bytes is
incorrect, and reversing runes alone can detach combining marks.

### Example: retrieve the segments of one mixed-direction line

This complete program uses real Hebrew characters, without `TestMode`. It prints
the information a client would pass to its rendering stage. `%+q` escapes
non-ASCII characters so the output can be inspected without a terminal applying
its own bidi display rules.

```go
package main

import (
	"fmt"
	"strings"

	"github.com/npillmayer/uax/bidi"
)

func main() {
	text := "Go אבג!"
	levels := bidi.ResolveParagraph(
		strings.NewReader(text), nil,
		bidi.DefaultDirection(bidi.LeftToRight),
	)

	// This paragraph fits on one line, so no Split is needed.
	order := levels.Reorder()
	for _, run := range order.Runs {
		for segment := range run.Segments(run.IsOpposite(bidi.LeftToRight)) {
			part := text[segment.From:segment.To]
			fmt.Printf("%s [%d:%d] %+q\n",
				segment.Dir, segment.From, segment.To, part)
			// Shape and draw part with direction segment.Dir here.
		}
	}
}
```

Output:

```text
L2R [0:3] "Go "
R2L [3:9] "\u05d0\u05d1\u05d2"
L2R [9:10] "!"
```

The Hebrew segment still contains its original logical character order. Its
three letters occupy six bytes, which explains the interval `[3:9]`.

### Inspecting results and supplying markup

Before reordering, `ResolvedLevels.Range()` provides a Go range iterator over
the outer resolved runs:

```go
for i, level := range levels.Range() {
	// level.BidiClass has type golang.org/x/text/unicode/bidi.Class.
	// level.From and level.To delimit a byte interval in the original text.
	fmt.Printf("%d: class=%v [%d:%d]\n",
		i, level.BidiClass, level.From, level.To)
}
```

Despite the name `Level`, this API exposes bidi classes and intervals, not
numeric embedding levels. It includes synthetic, zero-length outer LRI/RLI and
PDI delimiters and does not descend into nested isolates. Use
`LogicalSegments()` for nested directions before line breaking, and `Reorder()`
with the run segment iterators for output after line breaking.
`DirectionAt(pos)` likewise examines outer intervals only and falls back to
`LeftToRight` when no interval matches; it is not a complete nested-direction
query.

For directional boundaries supplied by external markup, pass an
`OutOfLineBidiMarkup` callback to `ResolveParagraph`. It receives a byte position
and should return `0` for no boundary, or one of `MarkupLRI`, `MarkupRLI`,
`MarkupPDI`, `MarkupPDILRI`, and `MarkupPDIRLI`. These constants are the callback's
encoding; do not return raw bidi class values. The inserted boundaries consume
zero bytes, preserving offsets into the client's text.

### Current implementation limits relevant to clients

Treat resolution, splitting, and reordering as a consuming pipeline. `Split`
and `Reorder` modify backing data. Copying their Go structs does not make
independent snapshots. Use `ResolvedLevels.Clone()` to make an independent
working copy before splitting or reordering. Inspect `Range()` and iterate
`LogicalSegments()` on the original paragraph result.

`ResolvedLevels.LogicalSegments` and `Run.Segments` do not modify their backing
data, including when `Run.Segments` iterates in reverse. Their iterators can be
reused, and a range loop can stop early with `break`.
The older `SegmentIterator` / `Next` / `Segment` API remains available for
compatibility, but `SegmentIterator(true)` reverses the run's backing segment
slice in place. Prefer `Segments` for new code, and avoid modifying a run while
iterating over it.

`Split(at, true)` is intended to rebase suffix positions to zero, for use with
a separately sliced text buffer. Currently it shifts only outer intervals,
not nested isolate children. Also, `Split` does not copy the paragraph's
embedding-direction field into its results. Splitting nested isolates or RTL
paragraphs therefore needs particular care; the simple single-line example
above avoids those unfinished paths.

By default, resolution stops at a paragraph separator. Supply a reader bounded
to one paragraph rather than relying on its position afterward: the scanner
may buffer ahead. `IgnoreParagraphSeparators(true)` treats separators as
whitespace; it does not provide multi-paragraph layout. `TestMode(true)` is for
algorithm tests: it treats uppercase letters as RTL and `>`, `<`, `=` as isolate
markers. Leave it off for normal text. `RecognizeLegacy` has no effect, and FSI
is not yet supported by the reordering implementation.

### Interaction with Text Shaping

Line breaking needs text widths, and those widths depend on the selected font
and shaping direction. First resolve bidi for the whole paragraph, then use
`LogicalSegments()` to intersect direction boundaries with font/style boundaries.
Do this before calling `Split` or `Reorder`. Unlike `Range()`, it descends into
nested isolates and returns a `Direction` directly. Number runs of bidi class
EN or AN are LTR even inside
an RTL isolate. The iterator omits zero-length intervals and isolate controls;
in-text controls leave gaps in the byte ranges, while external markup does not
consume bytes. Isolate boundaries are preserved even when adjacent segments
have the same direction. The result is not guaranteed to contain maximal runs
of a single direction.

This complete example prepares four shaping inputs by intersecting bidi
segments with three font spans. Only "FOUR" uses the bold font. The font names
stand for concrete fonts already selected by the client.
`TestMode(true)` makes uppercase letters represent RTL text for this example;
leave it off when processing real text.

```go
package main

import (
	"fmt"
	"strings"

	"github.com/npillmayer/uax/bidi"
)

func main() {
	text := "one two THREE FOUR FIVE"
	levels := bidi.ResolveParagraph(strings.NewReader(text), nil, bidi.TestMode(true))
	styles := []struct {
		from, to uint64
		font     string
	}{
		{0, 14, "regular"},
		{14, 18, "bold"},
		{18, uint64(len(text)), "regular"},
	}
	for segment := range levels.LogicalSegments() {
		for _, style := range styles {
			from := max(segment.From, style.from)
			to := min(segment.To, style.to)
			if from < to {
				// Shape text[from:to] using style.font and segment.Dir.
				fmt.Printf("%s %s [%d:%d] %q\n",
					segment.Dir, style.font, from, to, text[from:to])
			}
		}
	}
}
```

Output:

```text
L2R regular [0:8] "one two "
R2L regular [8:14] "THREE "
R2L bold [14:18] "FOUR"
R2L regular [18:23] " FIVE"
```

Replace the print operation with the client's shaping call, passing the selected
font, `segment.Dir`, and the original logical text `text[from:to]`. Do not reverse
the RTL input before shaping. Retain source-byte mappings with the glyphs and
advances so the line breaker and renderer can refer back to the original text.
If the shaper also requires uniform script or language, subdivide these items
at those boundaries as well.

`LogicalSegments()` leaves the resolved paragraph intact. Style boundaries
create shaping items; they do not require `levels.Split`. Once the line breaker
has selected actual line boundaries, split the resolved paragraph and call
`Reorder()` on each resulting line. `Run.Segments()` then identifies the source
intervals to associate with the shaped output for that line.

For larger inputs, the intersection can use a single forward scan of the two
ordered interval lists instead of the nested loops shown here.

To support repeated layout attempts, keep the original resolved paragraph and
clone it once per attempt. For the text above, a line breaker might select byte
offset 14 (before "FOUR") on its first attempt and 18 (after "FOUR") on another:

```go
for _, breakpoint := range []uint64{14, 18} {
	working := levels.Clone()
	firstLine, secondLine := working.Split(breakpoint, false)
	firstOrder := firstLine.Reorder()
	secondOrder := secondLine.Reorder()
	// Use firstOrder and secondOrder to place the shaped text for this attempt.
	_, _ = firstOrder, secondOrder
}
// levels.LogicalSegments() is still available for another shaping/layout pass.
```

`Clone()` copies the embedding direction and the entire nested isolate tree.
Its time and memory cost is proportional to the stored scraps and child slices;
it does not copy the client's text, fonts, or shaped glyphs. It copies the current
state, so cloning after `Reorder()` does not recover the original logical order.
Resolve again if the text or directional markup changes. Cloning preserves the
original result but does not change the `Split` limitations described above.
