package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

// This task can also be solve in two steps.
// 1. Fullfill Dir-Dir-File structure in memory
// 2. Print out that structure

// return a size of the file and return nothing if it is a directory
func getFileSize(fi os.FileInfo) string {
	if !fi.IsDir() {
		if fi.Size() == 0 {
			return " (empty)"
		}
		return " (" + strconv.FormatInt(fi.Size(), 10) + "b)"
	}
	return ""
}

func fileTree(out io.Writer, path string, printFiles bool, indent string) error {

	// Get list of files/dirs in path
	listFiles, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	// Get rid of files if printFiles set to true
	files := []os.FileInfo{}
	for _, file := range listFiles {
		if file.IsDir() || printFiles {
			files = append(files, file)
		}
	}

	// Sort
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Go through the file list
	i := 0
	for i < (len(files) - 1) {
		// Print line
		if printFiles || files[i].IsDir() {
			fmt.Fprint(out, indent, "├───", files[i].Name(), getFileSize(files[i]), "\n")
		}
		// If we've got a dir -> deep inside
		if files[i].IsDir() {
			fileTree(
				out,
				path+string(os.PathSeparator)+files[i].Name(),
				printFiles,
				indent+"│\t",
			)
		}
		i++
	}

	// Work with last one
	if len(files) > 0 {
		// Print line
		if printFiles || files[(len(files)-1)].IsDir() {
			fmt.Fprint(out, indent, "└───", files[(len(files)-1)].Name(), getFileSize(files[i]), "\n")
		}
		// If we've got a dir -> dive inside
		if files[(len(files) - 1)].IsDir() {
			fileTree(
				out,
				path+string(os.PathSeparator)+files[(len(files)-1)].Name(),
				printFiles,
				indent+"\t",
			)
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	// Start printing a dir structure without intend
	err := fileTree(out, path, printFiles, "")
	if err != nil {
		panic(err.Error())
	}
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"

	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
