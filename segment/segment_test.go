package segment

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/npillmayer/uax/internal/tracing"
)

func init() {
	corpusRunes = []rune(corpus)
}

func TestWhitespace1(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter()
	seg.Init(strings.NewReader("Hello World!"))
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment = '%s' with p = %d|%d", seg.Text(), p1, p2)
	}
}

func TestWhitespace2(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter()
	seg.Init(strings.NewReader("	for (i=0; i<5; i++)   count += i;"))
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment = '%s' with p = %d|%d", seg.Text(), p1, p2)
	}
}

func TestSimpleSegmenter1(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("Hello World "))
	n := 0
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		n++
	}
	if n != 4 {
		t.Errorf("Expected 4 segments, have %d", n)
	}
}
func TestSimpleSegmenter2(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("lime-tree"))
	n := 0
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		n++
	}
	if n != 1 {
		t.Errorf("Expected 1 segment, have %d", n)
	}
}

func TestBounded(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter(NewSimpleWordBreaker())
	seg.Init(strings.NewReader("Hello World, how are you?"))
	n := 0
	output := ""
	for seg.BoundedNext(14) {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		output += " [" + seg.Text() + "]"
		n++
	}
	t.Logf("seg.Err() = %v", seg.Err())
	t.Logf("seg.Text() = '%s'", seg.Text())
	t.Logf("bounded: output = %v", output)
	if n != 5 {
		t.Fatalf("Expected 5 segments, have %d", n)
	}
	t.Logf("bounded: passed 1st test ")
	tracing.Infof("======= rest =======")
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		output += " [" + seg.Text() + "]"
		n++
	}
	t.Logf("output = %v", output)
	if n != 10 {
		t.Errorf("Expected 10 segments, have %d", n)
	}
}

func TestBytesSegment(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter(NewSimpleWordBreaker())
	seg.Init(strings.NewReader("Hello World, how are you?"))
	n := 0
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("@ segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
		n++
	}
	if n != 9 {
		t.Errorf("Expected 9 segments, have %d", n)
	}
}

func TestRunesWriter(t *testing.T) {
	rw := runewrite{
		isBacked: true,
		backing:  []rune("Hello World!"),
	}
	rw = rw.SetMark()
	n, _ := (&rw).WriteRune('H')
	if n != 1 {
		t.Errorf("expected to have written 1 rune, but have %d", n)
	}
	s := rw.String()
	if s != "H" {
		t.Logf("s = %q", s)
		t.Error("expected written string to be \"H\", isn't")
	}
}

func TestRunesSlicing(t *testing.T) {
	tracing.SetTestingLog(t)
	//
	seg := NewSegmenter(NewSimpleWordBreaker())
	seg.InitFromSlice([]rune("Hello World, how are you?"))
	n := 0
	for seg.Next() {
		p1, p2 := seg.Penalties()
		t.Logf("@ segment: penalty = %5d|%d for breaking after '%v'\n",
			p1, p2, seg.Runes())
		n++
	}
	if n != 9 {
		t.Errorf("Expected 9 segments, have %d", n)
	}
}

func ExampleSegmenter() {
	seg := NewSegmenter() // will use a SimpleWordBreaker
	seg.Init(strings.NewReader("Hello World!"))
	for seg.Next() {
		p1, p2 := seg.Penalties()
		fmt.Printf("segment: penalty = %5d|%d for breaking after '%s'\n",
			p1, p2, seg.Text())
	}
	// Output:
	// segment: penalty =   100|0 for breaking after 'Hello'
	// segment: penalty =  -100|0 for breaking after ' '
	// segment: penalty =   100|0 for breaking after 'World!'
}

// --- Profiling -------------------------------------------------------------

var p1, p2 int

func BenchmarkBytesSegmenter(b *testing.B) {
	seg := NewSegmenter() // will use a SimpleWordBreaker
	for i := 0; i < b.N; i++ {
		corpusReader := strings.NewReader(corpus)
		seg.Init(corpusReader)
		for seg.Next() {
			p1, p2 = seg.Penalties()
		}
	}
}

func BenchmarkRunesSegmenter(b *testing.B) {
	seg := NewSegmenter() // will use a SimpleWordBreaker
	for i := 0; i < b.N; i++ {
		seg.InitFromSlice(corpusRunes)
		for seg.Next() {
			p1, p2 = seg.Penalties()
		}
	}
}

func BenchmarkScanSplit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		corpusReader := strings.NewReader(corpus)
		scan := bufio.NewScanner(corpusReader)
		scan.Split(bufio.ScanWords)
		for scan.Scan() {
			if len(scan.Bytes()) > 0 {
				p1 = len(scan.Bytes())
			}
		}
	}
}

var corpus = `
Im deutschen Grundgesetz ist der soziale Gedanke grundlegend verankert und sogar vor Änderungen geschützt. In politischen Diskussionen ist der Begriff bei uns durchgehend positiv besetzt, und dementsprechend wird er von Vertretern des gesamten politischen Spektrums vereinnahmt und gedeutet. Daran zeigt sich auch, dass der Begriff keineswegs einheitlich verstanden wird: Die soziale Gerechtigkeit des einen ist ungerecht aus Sicht des anderen.
Soziale Gerechtigkeit ist nicht gleichbedeutend mit vollständiger Gleichheit. In Deutschland folgen wir im Großen und Ganzen der Denkrichtung einer sozial-liberalen Gerechtigkeit, wie sie u.a. auf John Rawls zurück geht. Dabei akzeptieren wir Ungleichheiten, wie sie durch Glück, Leistung, Genetik usw. zustande kommen, bejahen aber auch ein Recht des Staats zur Umverteilung für gesamtgesellschaftliche Ziele.
Dieses Verständnis ist keineswegs universell; andere Gesellschaften akzentuieren den Gerechtigkeitsbegriff anders. Das angelsächsische Modell (USA, Großbritannien, Kanada, ...) verfolgt einen stärker liberitären Gedanken, während das skandinavische Modell (Schweden, Dänemark, Norwegen, ...) Gemeinschaft und Verteilung stärker betont [Merkel].
Soziale Gerechtigkeit steht auch im Spannungsfeld mit einem anderen hohen Gut: der persönlichen Freiheit. Bürger der USA betonen eher die Freiheit von Beeinträchtigungen, und empginden Umverteilung daher als etwas, das der Freiheit zuwiderläuft. Im sozial-liberalen Modell verstehen wir die Freiheit eher als Freiheit zu Handlungen, insbesondere der umfassenden Teilhabe am öffentlichen Leben. Dieser Freiheitsbegriff lässt sich leichter mit einem staatlichen Eingriff zur Umverteilung aussöhnen. Bei Zielkonglikten stimmen die meisten Bundesbürger „im Zweifel für die Freiheit“ [Freiheitsindex].
„Jede Gerechtigkeitstheorie fußt letzten Endes auf einer bestimmten Konzeption des erstrebenswerten Lebens in der Gemeinschaft und des angemessenen Gebrauchs unserer Freiheit, man könnte auch sagen auf einem bestimmten Menschenbild oder einer Vorstellung davon, worin die Würde des Menschen im Kern besteht. Darüber kann es in einer modernen pluralistischen Gesellschaft wohl keinen Konsens geben“ [Epbc9]. Das bedeutet, wir müssen immer wieder (im demokratischen Prozess) um eine Basis zur Verständigung ringen.
Geschichtliche Entwicklung
Mit dem Begriff der Gerechtigkeit befassten sich bereits Aristoteles und Platon. Für unser modernes Verständnis bahnbrechend war jedoch die Entwicklung der Idee individueller Freiheitsrechte gegenüber dem Staat im 16. Jahrhundert. Die Ständeordnung wich nach und nach anderen Gesellschaftsordnungen, in denen der Staat legitimiert werden musste, in die Freiheitsrechte des Einzelnen einzugreifen.
„Die bis dahin nicht in Zweifel gezogene Vorstellung, dass es so etwas wie ein objektives Gemeinwohl gibt, das im Erhalt des Ganzen besteht und sozusagen unabhängig vom Willen der Individuen vorgegeben ist, verliert an Bedeutung. Stattdessen beginnt man vielfach, das Gemeinwohl als Summe oder Querschnitt der Einzelinteressen zu verstehen, aus denen es in irgendeiner Weise abgeleitet werden muss“ [Epbc9].
Ein Ersatz der bis dahin vorausgesetzten göttlichen Ordnung kann durch das Gedankenexperiment eines Gesellschaftsvertrags gefunden werden. Insbesondere John Locke begründete ein liberales Gerechtigkeitsparadigma, das auf einem optimistischen Menschenbild beruht.
Die geburtsbedingte Zugehörigkeit von Individuen zu Gruppen (Ständen) wurde abgelöst durch eine durchlässige Verortung in sozialen Schichten. Ein Gerechtigkeitsverständnis, das die Arbeiterbewegungen bis heute prägt, geht auf Karl Marx (1818 – 1863) zurück. „Es hat sich ein traditionelles sozialdemokratisches Gerechtigkeitsparadigma herausgebildet, das sich vor allem durch Arbeitszentriertheit (gerechter Anspruch der Arbeiter auf das Arbeitsprodukt) und Klassenoder Kollektivzentriertheit (Gerechtigkeit für die ganze Klasse statt individueller Gerechtigkeit) auszeichnet“ [Epbc9].
Auch die katholische Kirche versuchte, ihren Beitrag zur Diskussion sozialer Gerechtigkeit zu leisten. 1891 und 1931 enstanden die päpstlichen Enzykliken, welche die katholische Soziallehre begründeten. „Eigentum verpglichtet“ ist das darin formulierte Leitmotiv, das sogar seinen Eingang in das Grundgesetz der Bundesrepublik fand (Artikel 14).
Einen der wichtigsten Beiträge zum Diskurs über soziale Gerechtigkeit lieferte der US-amerikanische Philosph John Rawls (1921 – 2002), der die Idee des Gesellschaftsvertrags neu augleben ließ und das Leitmotiv „Gerechtigkeit als Fairness“ zwischen Freien und Gleichen verfolgte. Daraus leitet Rawls zwei Grundsätze ab [Rawls]:
■ „Jedermann soll gleiches Recht auf das umfassendste System gleicher Grundfreiheiten haben, das mit dem gleichen System für alle anderen verträglich ist.“
■ „Soziale und wirtschaftliche Ungleichheiten sind so zu gestalten, dass (a) vernünftigerweise zu erwarten ist, dass sie zu jedermanns Vorteil dienen, und (b) sie mit Positionen und Ämtern verbunden sind, die jedem offenstehen.“
Rawls‘ Entwurf der Fairness ist einerseits liberal, erlaubt aber andererseits in gewissem Rahme eine Interpretation als Egalitarismus. Thomas Ebert schreibt dazu [Epbc9]:
a) Sein Egalitarismus bleibt immer liberal: Die Gleichheitsforderungen werden stets durch die absolut vorrangigen Freiheitsrechte begrenzt.
b) Sein Egalitarismus ist nur relativ und nicht absolut: Die Gleichheit ist kein Selbstzweck, sondern dient als Mittel zu dem Zweck, die Lage der Schwächsten zu verbessern; um dieses Zieles willen wird unter bestimmten Bedingungen auch Ungleichheit zugelassen.
Die Theorien von Rawls stehen in enger Verbindung mit dem Prinzip der Marktwirtschaft und sind daher für den zeitgenössische Diskurs besonders relevant. Der egalitäre Aspekt löste eine Gegenbewegung aus, die u.a. in der Doktrin des Neoliberalismus einen Ausdruck fand. Diese Strömung bezweifelt, dass soziale Gerechtigkeit überhaupt ein legitimes Ziel politischen Handelns ist. Hauptkritik an Spielarten des Egalitarismus ist die Feststellung, dass in einer pluralistischen Gesellschaft jedes Ziel einer Förderung bzw. eines ginanziellen Ausgleichs willkürlich sei1 und zwangsweise in einen „paternalistischen Verteilungsdespotismus“ münde [Mbcdbe]. Dies führe allenfalls zu einer degenerierten Gleichheit, in der – in Anlehnung an Orwells Roman „Animal Farm“ – manche eben „gleicher als andere“ seien.
`

var corpusRunes []rune
