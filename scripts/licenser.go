package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/offchainlabs/nitro/util/colors"
)

const currentYear = "2026"

var supportedExtensions = []string{".go", ".rs"}

func main() {
	flag.Parse()

	files, err := getFiles()
	if err != nil {
		exitWithError("could not list files: %v", err)
	}
	colors.PrintGrey(fmt.Sprintf("Found %d files", len(files)))

	for _, file := range files {
		if err = processFile(file); err != nil {
			exitWithError("could not process file %s: %v", file, err)
		}
	}
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

func exitWithError(format string, args ...interface{}) {
	colors.PrintRed("FATAL: ", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func processFile(path string) error {
	gitBirth, gitLast, err := getGitHistoryYears(path)
	if err != nil {
		return fmt.Errorf("could not get git years: %v", err)
	}
	colors.PrintGrey(fmt.Sprintf("[X] %-60s | Years: %s-%s", path, gitBirth, gitLast))
	return nil
}

func getGitHistoryYears(path string) (string, string, error) {
	// Get all years for this file, following renames, in chronological order
	// %ad = author date, --reverse puts the oldest commit first
	cmd := exec.Command("git", "log", "--follow", "--reverse", "--format=%ad", "--date=format:%Y", "--", path)
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("could not run git log: %v", err)
	}
	years := strings.Fields(string(out))

	if len(years) == 0 {
		return currentYear, currentYear, nil
	} else if len(years) == 1 {
		return years[0], years[0], nil
	}
	return years[0], years[len(years)-1], nil
}
