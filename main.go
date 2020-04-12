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
	// dst := os.Args[2]

	filepaths := walk(src)
	files := parseFiles(src, filepaths)
	files = validateReplacements(files)
	for _, file := range files {
		fmt.Println(file.path)
		for _, replacement := range file.replacements {
			fmt.Println("- ", replacement.path)
		}
	}
	// processFiles(src, dst, files)
	// writeManifest(dst, files)
}

func walk(src string) []string {
	files := []string{}
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

		files = append(files, path)
		return nil
	})
	return files
}

func parseFiles(src string, paths []string) []File {
	files := []File{}
	for _, path := range paths {
		// Add to queue
		relpath, err := filepath.Rel(src, path)
		if err != nil {
			panic(err)
		}
		ext := filepath.Ext(path)
		var replacements []Replacement
		if ext == ".css" {
			replacements = processCSS(path, relpath)
		} else {
			replacements = []Replacement{}
		}
		files = append(files, File{relpath, "", replacements})
	}
	return files
}

func validateReplacements(files []File) []File {
	// Construct a set of file paths
	paths := make(map[string]bool)
	for _, file := range files {
		paths[file.path] = true
	}

	// Loop through files, remove any non-resolving replacements
	for i, file := range files {
		newReplacements := []Replacement{}
		for _, replacement := range file.replacements {
			_, ok := paths[replacement.path]
			if ok {
				newReplacements = append(newReplacements, replacement)
			} else {
				fmt.Printf(
					"Warning: `%s` references nonexistant path `%s`\n",
					file.path,
					replacement.path,
				)
			}
		}
		file.replacements = newReplacements
		files[i] = file
	}
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
	srcFile, err := os.Open(filepath.Join(src, file.path))
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

	performReplacements(writer, reader, file, filemap)

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
	dstFn := filepath.Join(dst, file.hashedPath)
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

func performReplacements(writer io.Writer, reader io.Reader, file *File, filemap map[string]*File) {
	dir, _ := filepath.Split(file.path)
	lastPosition := 0
	for _, replacement := range file.replacements {
		toRead := replacement.position - lastPosition
		io.CopyN(writer, reader, int64(toRead))
		_, err := reader.Read(make([]byte, replacement.length))
		if err != nil {
			panic(err)
		}
		lastPosition = replacement.position + replacement.length

		refFile := filemap[replacement.path]
		refPath, err := filepath.Rel(dir, refFile.hashedPath)
		if err != nil {
			panic(err)
		}
		_, err = io.WriteString(writer, refPath)
		if err != nil {
			panic(err)
		}
	}
	_, err := io.Copy(writer, reader)
	if err != nil {
		panic(err)
	}
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
