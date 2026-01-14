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
	gitBirth, err := getGitBirthYear(path)
	if err != nil {
		return fmt.Errorf("could not get birth year: %v", err)
	}
	gitLast, err := getGitLastUpdateYear(path)
	if err != nil {
		return fmt.Errorf("could not get last update year: %v", err)
	}

	colors.PrintGrey(fmt.Sprintf("[X] %-60s | Years: %s-%s", path, gitBirth, gitLast))
	return nil
}

func getGitBirthYear(path string) (string, error) {
	cmd := exec.Command("git", "log", "--follow", "--diff-filter=A", "--format=%ad", "--date=format:%Y", "--", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	years := strings.Fields(string(output))
	if len(years) > 0 {
		return years[len(years)-1], nil
	}
	// Fallback to current year if no commits found
	return "2026", nil
}

func getGitLastUpdateYear(path string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ad", "--date=format:%Y", "--", path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	last := strings.TrimSpace(string(output))
	if last == "" {
		// Fallback to current year if no commits found
		return "2026", nil
	}
	return last, nil
}
