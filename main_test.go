package main

import (
	"crypto/md5"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestWalk(t *testing.T) {
	dir, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(dir)

	// Directory structure:
	// foo.jpg
	// fizz
	// > buzz.jpg
	// > .ignore
	ioutil.WriteFile(filepath.Join(dir, "foo.jpg"), []byte(""), 0644)
	os.Mkdir(filepath.Join(dir, "fizz"), 0755)
	ioutil.WriteFile(filepath.Join(dir, "fizz/buzz.jpg"), []byte(""), 0644)
	ioutil.WriteFile(filepath.Join(dir, "fizz/.ignore"), []byte(""), 0644)

	files := walk(dir)
	sort.Strings(files)
	assert.Equal(t, []string{"fizz/buzz.jpg", "foo.jpg"}, files)
}

func TestParseFiles(t *testing.T) {
	dir, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(dir)

	css := "url('foo.jpg')"

	ioutil.WriteFile(filepath.Join(dir, "foo.jpg"), []byte("nothing"), 0644)
	ioutil.WriteFile(filepath.Join(dir, "bar.css"), []byte(css), 0644)

	files := parseFiles(dir, []string{"foo.jpg", "bar.css"})
	assert.Equal(t, 2, len(files))

	f1 := files[0]
	assert.Equal(t, "foo.jpg", f1.path)
	assert.Equal(t, 0, len(f1.replacements))

	f2 := files[1]
	assert.Equal(t, "bar.css", f2.path)
	assert.Equal(t, 1, len(f2.replacements))
	assert.Equal(t, Replacement{
		position: 5,
		length:   7,
		path:     "foo.jpg",
	}, f2.replacements[0])
}

func TestValidateReplacements(t *testing.T) {
	// Set up foo.jpg, and bar.css which references foo.jpg and a nonexistent file
	r1 := Replacement{
		position: 10,
		length:   10,
		path:     "foo.jpg",
	}
	r2 := Replacement{
		position: 20,
		length:   5,
		path:     "fizz.jpg",
	}
	barcss := File{
		path:         "bar.css",
		replacements: []Replacement{r1, r2},
	}
	foojpg := File{
		path:         "foo.jpg",
		replacements: []Replacement{},
	}
	files := []File{barcss, foojpg}

	newfiles := validateReplacements(files)
	assert.Equal(t, 2, len(newfiles))
	assert.Equal(t, "bar.css", newfiles[0].path)
	assert.Equal(t, 1, len(newfiles[0].replacements))
	assert.Equal(t, r1, newfiles[0].replacements[0])
	assert.Equal(t, foojpg, newfiles[1])
}

func TestIsReady(t *testing.T) {
	file := File{
		replacements: []Replacement{
			Replacement{path: "foo.jpg"},
			Replacement{path: "bar.jpg"},
		},
	}
	filemap := map[string]string{
		"foo.jpg":  "",
		"fizz.jpg": "",
	}
	assert.Equal(t, false, isReady(file, filemap))
	filemap["bar.jpg"] = ""
	assert.Equal(t, true, isReady(file, filemap))
}

func TestProcessFiles(t *testing.T) {
	src, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(src)
	dst, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(dst)

	css := "url('foo.jpg')"
	ioutil.WriteFile(filepath.Join(src, "bar.css"), []byte(css), 0644)
	ioutil.WriteFile(filepath.Join(src, "foo.jpg"), []byte("jpeg"), 0644)
	cssfile := File{
		path: "bar.css",
		replacements: []Replacement{
			Replacement{
				position: 5,
				length:   7,
				path:     "foo.jpg",
			},
		},
	}
	files := []File{
		cssfile,
		File{path: "foo.jpg", replacements: []Replacement{}},
	}

	filemap := processFiles(src, dst, files)
	assert.Equal(t, map[string]string{
		"foo.jpg": "foo.ab4f3ccb.jpg",
		"bar.css": "bar.46323437.css",
	}, filemap)
	content, _ := ioutil.ReadFile(filepath.Join(dst, "foo.ab4f3ccb.jpg"))
	assert.Equal(t, []byte("jpeg"), content)
	content, _ = ioutil.ReadFile(filepath.Join(dst, "bar.46323437.css"))
	assert.Equal(t, []byte("url('foo.ab4f3ccb.jpg')"), content)
}

func TestProcessFile(t *testing.T) {
	src, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(src)
	dst, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(dst)

	css := "url('foo.jpg')"
	ioutil.WriteFile(filepath.Join(src, "bar.css"), []byte(css), 0644)
	file := File{
		path: "bar.css",
		replacements: []Replacement{
			Replacement{
				position: 5,
				length:   7,
				path:     "foo.jpg",
			},
		},
	}
	filemap := map[string]string{
		"foo.jpg": "foo.ab4f3ccb.jpg",
	}

	fn := processFile(src, dst, file, filemap)
	assert.Equal(t, "bar.46323437.css", fn)
	content, _ := ioutil.ReadFile(filepath.Join(dst, fn))
	assert.Equal(t, []byte("url('foo.ab4f3ccb.jpg')"), content)
}

func TestPerformReplacementsNone(t *testing.T) {
	var writer strings.Builder
	reader := strings.NewReader("content")
	file := File{
		path:         "foo.jpg",
		replacements: []Replacement{},
	}
	filemap := map[string]string{"fizz": "buzz"}
	performReplacements(&writer, reader, file, filemap)
	assert.Equal(t, "content", writer.String())
}

func TestPerformReplacements(t *testing.T) {
	var writer strings.Builder
	reader := strings.NewReader("foobarfuzz")
	file := File{
		path: "foo.jpg",
		replacements: []Replacement{
			Replacement{
				path:     "fizz",
				position: 3,
				length:   3,
			},
			Replacement{
				path:     "1",
				position: 8,
				length:   1,
			},
		},
	}
	filemap := map[string]string{
		"fizz": "buzz",
		"1":    "2",
	}
	performReplacements(&writer, reader, file, filemap)
	assert.Equal(t, "foobuzzfu2z", writer.String())
}

func TestCreateHashedFilename(t *testing.T) {
	hash := md5.Sum([]byte("test"))
	fn := createHashedFilename("dir/main.css", hash[:])
	assert.Equal(t, "dir/main.098f6bcd.css", fn)
}

func TestWriteManifest(t *testing.T) {
	dir, _ := ioutil.TempDir("", "hashthing")
	defer os.RemoveAll(dir)
	fn := filepath.Join(dir, "manifest.json")
	filemap := map[string]string{
		"foo.jpg":  "foo.abc.jpg",
		"fizz.jpg": "fizz.def.jpg",
	}
	writeManifest(fn, filemap)
	expected := `{
  "fizz.jpg": "fizz.def.jpg",
  "foo.jpg": "foo.abc.jpg"
}`
	contents, _ := ioutil.ReadFile(fn)
	assert.Equal(t, expected, string(contents))
}
