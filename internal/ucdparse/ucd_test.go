package ucdparse

import (
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	input := strings.NewReader("000E..001F;CM     # Cc    [18] <control-000E>..<control-001F>")
	sc, err := New(input)
	if err != nil {
		t.Fatal(err)
	}
	if !sc.Next() {
		t.Logf("token = %v", sc.Token)
		t.Fatal(sc.Token.Error)
	}
	if !sc.Next() {
		t.Logf("token = %v", sc.Token)
		t.Fatal(sc.Token.Error)
	}
	t.Logf("token = %v", sc.Token)
	if sc.Token.Field(1) != "CM" {
		t.Errorf("expected field #1 to be 'CM', is %q", sc.Token.Field(1))
	}
	from, to := sc.Token.Range()
	if from != 0x0e || to != 0x1f {
		t.Errorf("expected range to be 0E..1F, is %02X..%02X", from, to)
	}
}
