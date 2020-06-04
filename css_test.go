package main

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestProcessCSS(t *testing.T) {
	content := `
body {
	background: url("foo.jpg");
}
div {
	background: url("../fizz/buzz.jpg");
}
`
	file, _ := ioutil.TempFile("", "hashthing")
	defer os.Remove(file.Name())
	file.Write([]byte(content))
	file.Seek(0, 0)
	replacements := processCSS(file.Name(), "foo/bar.css")
	assert.Equal(t, 2, len(replacements))
	r1 := replacements[0]
	assert.Equal(t, "foo/foo.jpg", r1.path)
	assert.Equal(t, "foo.jpg", content[r1.position:r1.position+r1.length])
	r2 := replacements[1]
	assert.Equal(t, "fizz/buzz.jpg", r2.path)
	assert.Equal(t, "../fizz/buzz.jpg", content[r2.position:r2.position+r2.length])
}

func TestReadURL(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("url(foo.jpg)"))
	offset, path := readURL(reader)
	assert.Equal(t, 4, offset)
	assert.Equal(t, "foo.jpg", path)
	// Ensure the parenthesis is remaining
	remaining := make([]byte, 1)
	reader.Read(remaining)
	assert.Equal(t, ")", string(remaining))
}

func TestReadURLExtraChars(t *testing.T) {
	// This is completely invalid CSS, but our parser doesn't care.
	reader := bufio.NewReader(strings.NewReader("url( \t \"foo.jpg\"morestuff"))
	offset, path := readURL(reader)
	assert.Equal(t, 8, offset)
	assert.Equal(t, "foo.jpg", path)
	// Ensure the parenthesis is remaining
	remaining := make([]byte, 10)
	reader.Read(remaining)
	assert.Equal(t, "\"morestuff", string(remaining))
}

func TestMakeRelPath(t *testing.T) {
	path, valid := makeRelPath("foo/", "foo/bar.jpg")
	assert.Equal(t, true, valid)
	assert.Equal(t, "foo/foo/bar.jpg", path)
	path, valid = makeRelPath("foo/", "/foo/bar.jpg")
	assert.Equal(t, false, valid)
	path, valid = makeRelPath("foo/", "http://example.com/foo/bar.jpg")
	assert.Equal(t, false, valid)
	path, valid = makeRelPath("foo/", "foo/bar%20.jpg?test=foo#fizz=buzz")
	assert.Equal(t, true, valid)
	assert.Equal(t, "foo/foo/bar .jpg", path)
}

func TestByteInArray(t *testing.T) {
	assert.Equal(t, true, byteInArray('a', []byte("abc")))
	assert.Equal(t, false, byteInArray('d', []byte("abc")))
}

func TestReadwhile(t *testing.T) {
	reader := strings.NewReader("url(foo")
	bufReader := bufio.NewReader(reader)
	r := readWhile(bufReader, func(b byte) bool { return b != '(' })
	assert.Equal(t, "url", string(r))
	p, _ := bufReader.Peek(4)
	assert.Equal(t, "(foo", string(p))
}
