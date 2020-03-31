package main

import (
	"bufio"
	// "bytes"
	"fmt"
	"strings"
	"os"
	"path/filepath"
	"io"
	"io/ioutil"
	"crypto/md5"
	"encoding/json"
)

func main() {
	if len(os.Args) <= 2 {
		fmt.Println("Must include source and destination directories.")
		os.Exit(1)
	}
	src := os.Args[1]
	dst := os.Args[2]

	files := walk(src)
	processFiles(src, dst, files)
	writeManifest(dst, files)
}

func walk(src string) map[string]*File {
	files := make(map[string]*File)
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		// Ignore directories
		fi, err := os.Stat(path)
		if err != nil {
			panic(err)
		}
		if fi.IsDir() {
			return nil
		}

		// Ignore dotted files
		_, file := filepath.Split(path)
		if file[0] == "."[0] {
			return nil
		}

		// Add to queue
		ext := filepath.Ext(path)
		var replacements []Replacement
		if ext == ".css" {
			replacements = processCSS(path)
		} else {
			replacements = []Replacement{}
		}
		files[path] = &File{path, "", replacements}
		return nil
	})
	return files
}

func processFiles(src string, dst string, files map[string]*File) {
	for {
		processed := 0
		for _, file := range files {
			if file.hashedPath == "" && isReady(*file, files) {
				processFile(src, dst, file, files)
				processed += 1
			}
		}
		if processed == 0 {
			break
		}
	}
}

func isReady(file File, files map[string]*File) bool {
	for _, replacement := range file.replacements {
		ref, ok := files[replacement.path]
		if !ok {
			fmt.Println(
				"Non-existant file `%s` referenced in `%s`",
				replacement.path,
				file.path,
			)
			panic("omg")
		}
		if ref.hashedPath == "" {
			return false
		}
	}
	return true
}

func processFile(src string, dst string, file *File, filemap map[string]*File) {
	// Read the file
	fmt.Println("Processing %s", file.path)
	dir, fn := filepath.Split(file.path)

	// Open file for reading
	srcFile, err := os.Open(file.path)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()
	reader := bufio.NewReader(srcFile)

	// Open temp file for writing
	dstFile, err := ioutil.TempFile("", "hashthing")
	if err != nil {
		panic(err)
	}
	dstBuffer := bufio.NewWriter(dstFile)

	// Create hash
	hash := md5.New()
	writer := io.MultiWriter(hash, dstBuffer)

	lastPosition := 0
	for _, replacement := range file.replacements {
		toRead := replacement.position - lastPosition
		io.CopyN(writer, reader, int64(toRead))
		reader.Discard(replacement.length)
		lastPosition = replacement.position + replacement.length

		refFile := filemap[replacement.path]
		whatever, err := filepath.Rel(dir, refFile.hashedPath)
		if err != nil {
			panic(err)
		}
		fmt.Println(whatever)
		_, err = io.WriteString(writer, whatever)
		if err != nil {
			panic(err)
		}
	}
	_, err = io.Copy(writer, reader)
	if err != nil {
		panic(err)
	}
	err = srcFile.Close()
	if err != nil {
		panic(err)
	}
	err = dstBuffer.Flush()
	if err != nil {
		panic(err)
	}
	err = dstFile.Close()
	if err != nil {
		panic(err)
	}

	// Create hashed filename
	hashString := fmt.Sprintf("%x", hash.Sum(nil))[:8]
	file.hashedPath = filepath.Join(dir, createFilename(fn, hashString))

	// Write the file
	relpath, err := filepath.Rel(src, file.hashedPath)
	dstFn := filepath.Join(dst, relpath)
	fmt.Println(dstFn)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(filepath.Dir(dstFn), 0755)
	if err != nil {
		panic(err)
	}
	os.Rename(dstFile.Name(), dstFn)
}

func createFilename(fn string, hash string) string {
	fnSplit := strings.Split(fn, ".")
	newFn := make([]string, len(fnSplit) - 1, len(fnSplit) + 1)
	copy(newFn, fnSplit[:len(fnSplit) - 1])
	newFn = append(newFn, hash, fnSplit[len(fnSplit) - 1])
	return strings.Join(newFn, ".")
}

func writeManifest(src string, files map[string]*File) {
	manifest := make(map[string]string)
	for _, file := range files {
		manifest[file.path] = file.hashedPath
	}
	fn := filepath.Join(src, "manifest.json")
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(fn, contents, 0644)
}

type File struct {
	path string
	hashedPath string
	replacements []Replacement
}

type Replacement struct {
	position int
	length int
	path string
}
