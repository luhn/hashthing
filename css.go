package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// https://www.w3.org/TR/CSS1/#url

// PATTERN = regexp.MustCompile("url")

func processCSS(fullpath string, path string) []Replacement {
	dir := filepath.Dir(path)
	fh, err := os.Open(fullpath)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	reader := bufio.NewReader(fh)

	// Scan
	pos := 0
	replacements := []Replacement{}

mainLoop:
	for {
		for {
			b, err := reader.Peek(4)
			if err != nil {
				if err.Error() == "EOF" {
					break mainLoop
				} else {
					panic(err)
				}
			}
			if bytes.Compare(b, []byte("url(")) == 0 {
				offset, rawpath := readURL(reader)
				if isValidPath(rawpath) {
					relpath := filepath.Join(dir, rawpath)
					replacements = append(replacements, Replacement{
						position: pos + offset,
						length:   len(rawpath),
						path:     relpath,
					})
					pos += offset + len(path)
				}
			} else {
				reader.Discard(1)
				pos += 1
			}
		}
	}
	return replacements
}

func readURL(reader *bufio.Reader) (int, string) {
	// Returns offset, path

	// Start by discarding "url("
	reader.Discard(4)
	offset := 4

	// Discard starting characters
	STARTCHARS := []byte(" \t\"'")
	discarded := readWhile(reader, func(b byte) bool {
		return byteInArray(b, STARTCHARS)
	})
	offset += len(discarded)

	ENDCHARS := []byte(" \t)\"'")
	path := readWhile(reader, func(b byte) bool {
		return !byteInArray(b, ENDCHARS)
	})

	return offset, string(path)
}

func isValidPath(path string) bool {
	// No absolute paths
	if path[0:1] == "/" {
		return false
	}
	// No URLs
	if strings.Contains(path, "://") {
		return false
	}
	// Otherwise okay
	return true
}

// Util functions for parsing file

func byteInArray(needle byte, haystack []byte) bool {
	for _, b := range haystack {
		if b == needle {
			return true
		}
	}
	return false
}

func readWhile(reader *bufio.Reader, cond func(byte) bool) []byte {
	var output []byte
	for {
		bytes, err := reader.Peek(1)
		if err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				panic(err)
			}
		}
		b := bytes[0]
		if cond(b) {
			reader.Discard(1)
			output = append(output, b)
		} else {
			break
		}
	}
	return output
}
