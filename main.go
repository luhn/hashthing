package main

import (
	"fmt"
	"strings"
	"os"
	"path/filepath"
	"io/ioutil"
	"crypto/md5"
)

func main() {
	if len(os.Args) <= 2 {
		fmt.Println("Must include source and destination directories.")
		os.Exit(1)
	}
	src := os.Args[1]
	dst := os.Args[2]

	files := walk(src)
	queue := NewQueue(files)
	processQueue(src, dst, queue)
	// fmt.Println(files)
}

func walk(src string) []File {
	files := make([]File, 0, 256)
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		relpath, err := filepath.Rel(src, path)
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
		_, file := filepath.Split(relpath)
		if file[0] == "."[0] {
			return nil
		}

		// Add to queue
		files = append(files, File{relpath, nil})
		return nil
	})
	return files
}

func processQueue(src string, dst string, files FileQueue) {
	for !files.Empty() {
		file := files.Pop()
		if len(file.replacements) > 0 {
			panic("We don't support replacements yet!")
		}
		fmt.Println(file)
		processFile(src, dst, file)
	}
}

func processFile(src string, dst string, file File) {
	// Read the file
	data, err := ioutil.ReadFile(filepath.Join(src, file.path))
	if err != nil {
		panic(err)
	}

	// Create the new name
	dir, fn := filepath.Split(filepath.Join(dst, file.path))
	hash := fmt.Sprintf("%x", md5.Sum(data))[:8]
	dstPath := filepath.Join(dir, createFilename(fn, hash))

	// Write the file
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(dstPath, data, 0644)
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

type File struct {
	path string
	replacements []Replacement
}

type Replacement struct {
	position int
	length int
	path string
}
