package main

import (
	"bufio"
	// "bytes"
	"flag"
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
	manifest := flag.String(
		"manifest",
		"manifest.json",
		"The file path to output the JSON manifest file.",
	)
	flag.Usage = func() {
		fmt.Print(
			"Usage: hashthing [options] src dst\n\n" +
			"Hashthing will recursively iterate through `src`, append a hash of " +
			"the file to the filename, and copy to `dst`.  Relative paths in CSS " +
			"documents will be rewritten to include the hash.\n\n" +
			"`src` and `dst` should be directories.  If `dst` does not exist, it " +
			"will be created.\n\n" +
			"For more information, visit https://github.com/luhn/hashthing\n\n")
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("Must include source and destination directories.")
		os.Exit(1)
	} else if flag.NArg() > 2 {
		fmt.Printf("Expected 2 arguments, received %d.\n", flag.NArg())
		os.Exit(1)
	}
	src := flag.Arg(0)
	dst := flag.Arg(1)

	filepaths := walk(src)
	files := parseFiles(src, filepaths)
	files = validateReplacements(files)
	for _, file := range files {
		fmt.Println(file.path)
		for _, replacement := range file.replacements {
			fmt.Println("- ", replacement.path)
		}
	}
	// Create dst directory, if not exists
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		panic(err)
	}
	processed := processFiles(src, dst, files)
	writeManifest(*manifest, processed)
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

func processFiles(src string, dst string, files []File) map[string]string {
	processed := make(map[string]string)
	for len(files) > 0 {
		var file File
		file, files = files[len(files)-1], files[:len(files)-1]
		if isReady(file, processed) {
			processed[file.path] = processFile(src, dst, file, processed)
		} else {
			files = append(files, file)
		}
	}
	return processed
}

func isReady(file File, processed map[string]string) bool {
	for _, replacement := range file.replacements {
		_, ok := processed[replacement.path]
		if !ok {
			return false
		}
	}
	return true
}

func processFile(src string, dst string, file File, filemap map[string]string) string {
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
	// We're creating a temp file in the dst directory, rather than the default
	// temp directory, because the dst directory may be on another different
	// volume than the temp directory.  This would break os.Rename below.
	dstFile, err := ioutil.TempFile(dst, "hashthing")
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
	hashedPath := filepath.Join(dir, createHashedFilename(fn, hash.Sum(nil)))

	// Move to final home
	dstPath := filepath.Join(dst, hashedPath)
	err = os.MkdirAll(filepath.Dir(dstPath), 0755)
	if err != nil {
		panic(err)
	}
	os.Rename(dstFile.Name(), dstPath)

	return hashedPath
}

func performReplacements(writer io.Writer, reader io.Reader, file File, filemap map[string]string) {
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
		refPath, err := filepath.Rel(dir, refFile)
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

func createHashedFilename(fn string, hash []byte) string {
	hashString := fmt.Sprintf("%x", hash)[:8]
	fnSplit := strings.Split(fn, ".")
	newFn := make([]string, len(fnSplit) - 1, len(fnSplit) + 1)
	copy(newFn, fnSplit[:len(fnSplit) - 1])
	newFn = append(newFn, hashString, fnSplit[len(fnSplit) - 1])
	return strings.Join(newFn, ".")
}

func writeManifest(path string, filemap map[string]string) {
	contents, err := json.MarshalIndent(filemap, "", "  ")
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(path, contents, 0644)
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
