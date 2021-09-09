package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pkg/errors"
)

type Files []os.FileInfo

func (s Files) Len() int           { return len(s) }
func (s Files) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Files) Less(i, j int) bool { return s[i].Name() < s[j].Name() }

func readDir(path string, printFiles bool) (Files, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Errorf("Error while reading path %s: %#v", path, err)
	}
	files, err := file.Readdir(0)
	if err != nil {
		return nil, errors.Errorf("Error while reading file contents %s: %#v", path, err)
	}
	filtered := Files{}
	for _, file := range files {
		if file.IsDir() {
			filtered = append(filtered, file)
		} else if printFiles {
			filtered = append(filtered, file)
		}
	}

	sort.Sort(filtered)
	return filtered, nil
}

func writeDir(out io.Writer, path string, printFiles bool, files Files, prefix string) error {
	for i, file := range files {
		out.Write([]byte(prefix))
		var newPrefix string
		if i < len(files)-1 {
			out.Write([]byte("├───"))
			newPrefix = prefix + "│\t"
		} else {
			out.Write([]byte("└───"))
			newPrefix = prefix + "\t"
		}
		out.Write([]byte(file.Name()))
		if !file.IsDir() {
			out.Write([]byte(" ("))
			if file.Size() == 0 {
				out.Write([]byte("empty"))
			} else {
				out.Write([]byte(fmt.Sprintf("%db", file.Size())))
			}
			out.Write([]byte(")"))
		}
		out.Write([]byte("\n"))
		if file.IsDir() {
			folderFiles, err := readDir(path+string(os.PathSeparator)+file.Name(), printFiles)
			if err != nil {
				return err
			}
			if len(folderFiles) > 0 {
				err := writeDir(out, path+string(os.PathSeparator)+file.Name(), printFiles, folderFiles, newPrefix)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	files, err := readDir(path, printFiles)
	if err != nil {
		return err
	}
	return writeDir(out, path, printFiles, files, "")
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
