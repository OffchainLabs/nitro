package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"
)

var supportedExtensions = []string{".go", ".rs"}

func main() {
	flag.Parse()

	files, err := getFiles()
	if err != nil {
		fmt.Printf("Fatal: could not list files: %v\n", err)
		return
	}
	fmt.Printf("Found %d files\n", len(files))
}

func getFiles() ([]string, error) {
	out, err := exec.Command("git", "ls-files").Output()
	if err != nil {
		return nil, err
	}
	var filtered []string
	for _, line := range strings.Split(string(out), "\n") {
		for _, extension := range supportedExtensions {
			if strings.HasSuffix(line, extension) {
				filtered = append(filtered, line)
			}
		}
	}
	return filtered, nil
}
