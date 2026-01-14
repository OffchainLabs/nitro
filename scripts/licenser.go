package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/offchainlabs/nitro/util/colors"
)

var supportedExtensions = []string{".go", ".rs"}

func main() {
	flag.Parse()

	files, err := getFiles()
	if err != nil {
		colors.PrintRed("Fatal: could not list files: ", err)
		os.Exit(1)
	}
	colors.PrintGrey(fmt.Sprintf("Found %d files", len(files)))
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
