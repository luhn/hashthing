package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// Perform simple parsing on the CSS file at `fullpath` to discover any file
// references, i.e. `url()`.  Parsing rules are derived from the CSS1 spec:
// https://www.w3.org/TR/CSS1/#url
//
// `path` is the path of the CSS file relative to the source directory.  This
// is used to determine the path of referenced files relative to the source
// directory.
//
// Only relative paths are kept, absolute paths and URLs are discarded.
func processCSS(fullpath string, path string) []Replacement {
	dir := filepath.Dir(path)
	fh, err := os.Open(fullpath)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	reader := bufio.NewReader(fh)

	pos := 0 // Our current position in the file.
	replacements := []Replacement{}

mainLoop:
	for {
		for {
			// Look ahead four bytes for `url(`
			b, err := reader.Peek(4)
			if err != nil {
				if err.Error() == "EOF" {
					break mainLoop
				} else {
					panic(err)
				}
			}
			if bytes.Compare(b, []byte("url(")) == 0 {
				// When `url(` is hit, read the path and add a Replacement
				offset, rawpath := readURL(reader)
				if isValidPath(rawpath) {
					relpath := filepath.Join(dir, rawpath)
					replacements = append(replacements, Replacement{
						position: pos + offset,
						length:   len(rawpath),
						path:     relpath,
					})
					pos += offset + len(rawpath)
				}
			} else {
				// Move forward to the next space.
				reader.Discard(1)
				pos += 1
			}
		}
	}
	return replacements
}

// Called once `url(` is hit.  Discards all superfluous characters and returns
// the path.
func readURL(reader *bufio.Reader) (int, string) {
	// Discard `url(`
	start := make([]byte, 4)
	reader.Read(start)
	if string(start) != "url(" {
		panic("readURL called at the wrong place!")
	}
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

// Determine if a path is valid.  Absolute paths and URLs are invalid.
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

/* Util functions for parsing file */

// Return true if byte array `haystack` contains byte `needle`.
func byteInArray(needle byte, haystack []byte) bool {
	for _, b := range haystack {
		if b == needle {
			return true
		}
	}
	return false
}

// Read bytes from the reader while the given condition holds.  Returns all
// read bytes.
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
