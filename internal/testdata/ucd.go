package testdata

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// UCDReader returns reader for the given ucd file for testing.
func UCDReader(file string) (io.Reader, error) {
	data, err := os.ReadFile(UCDPath(file))
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

// UCDPath returns path for the given ucd file.
func UCDPath(file string) string {
	_, pkgdir, _, ok := runtime.Caller(0)
	if !ok {
		panic("no debug info")
	}

	return filepath.Join(filepath.Dir(pkgdir), "ucd", file)
}
