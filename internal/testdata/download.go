// +build ignore

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	err := downloadUCDZip("https://www.unicode.org/Public/11.0.0/ucd/UCD.zip", "ucd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to download: %v\n", err)
		os.Exit(1)
	}
}

func downloadUCDZip(url, dir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("GET failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	for _, file := range z.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name == "auxiliary/LineBreakTest.txt" {
			// LineBreakTest.txt has skipped tests.
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open %v: %w", file.Name, err)
		}
		if err := writeFile(filepath.Join(dir, file.Name), rc); err != nil {
			return fmt.Errorf("failed to write %v: %w", file.Name, err)
		}
	}

	return nil
}

func writeFile(path string, rc io.ReadCloser) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	defer func() { _ = rc.Close() }()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %v: %w", path, err)
	}

	_, err = io.Copy(f, rc)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to copy %v: %w", path, err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("failed to write %v: %w", path, err)
	}

	return nil
}
