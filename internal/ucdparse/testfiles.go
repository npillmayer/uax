package ucdparse

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

type ucdTestFile struct {
	in      *os.File
	scanner *bufio.Scanner
	text    string
	comment string
}

func OpenTestFile(filename string, t *testing.T) *ucdTestFile {
	f, err := os.Open(filename)
	if err != nil {
		if t != nil {
			t.Errorf("ERROR loading " + filename)
		} else {
			fmt.Fprintf(os.Stderr, "ERROR loading "+filename)
		}
		return nil
	}
	tf := &ucdTestFile{}
	tf.in = f
	tf.scanner = bufio.NewScanner(f)
	return tf
}

func (tf *ucdTestFile) Scan() bool {
	ok := true
	done := false
	for !done {
		ok = tf.scanner.Scan()
		if ok && len(tf.scanner.Bytes()) > 0 {
			if tf.scanner.Bytes()[0] == '#' {
				continue
			}
			done = true
			text := tf.scanner.Text()
			text = strings.TrimSpace(text)
			parts := strings.Split(text, "#")
			if len(parts) > 1 {
				tf.text, tf.comment = parts[0], parts[1]
			} else {
				tf.text = parts[0]
			}
		} else {
			done = true // with error
		}
	}
	return ok
}

func (tf *ucdTestFile) Text() string {
	return tf.text
}

func (tf *ucdTestFile) Comment() string {
	return tf.comment
}

func (tf *ucdTestFile) Err() error {
	return tf.scanner.Err()
}

func (tf *ucdTestFile) Close() {
	tf.in.Close()
}

func BreakTestInput(ti string) (string, []string) {
	//fmt.Printf("breaking up %s\n", ti)
	sc := bufio.NewScanner(strings.NewReader(ti))
	sc.Split(bufio.ScanWords)
	out := make([]string, 0, 5)
	inp := bytes.NewBuffer(make([]byte, 0, 20))
	run := bytes.NewBuffer(make([]byte, 0, 20))
	for sc.Scan() {
		token := sc.Text()
		if token == "÷" {
			if run.Len() > 0 {
				out = append(out, run.String())
				run.Reset()
			}
		} else if token == "×" {
			// do nothing
		} else {
			n, _ := strconv.ParseUint(token, 16, 64)
			run.WriteRune(rune(n))
			inp.WriteRune(rune(n))
		}
	}
	//fmt.Printf("input = '%s'\n", inp.String())
	//fmt.Printf("output = %#v\n", out)
	return inp.String(), out
}
