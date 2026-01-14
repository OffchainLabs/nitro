package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/offchainlabs/nitro/util/colors"
)

const (
	currentYear = "2026"
	company     = "Offchain Labs, Inc."
	licenseURL  = "https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md"
)

var (
	supportedExtensions = []string{".go", ".rs"}
	yearRegex           = regexp.MustCompile(`Copyright\s+(\d{4})(?:-(\d{4}))?`)
	fixFlag             = flag.Bool("fix", false, "Update files with the correct license header")
)

type Stats struct {
	Total int
	Valid int
	Fixed int
}

func main() {
	flag.Parse()

	files, err := getFiles()
	if err != nil {
		exitWithError("could not list files: %v", err)
	}

	stats := &Stats{Total: len(files)}
	for _, file := range files {
		if err = processFile(file, stats); err != nil {
			exitWithError("could not process file %s: %v", file, err)
		}
	}
	printSummary(stats)
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
				break
			}
		}
	}
	return filtered, nil
}

func exitWithError(format string, args ...interface{}) {
	colors.PrintRed("FATAL: ", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func processFile(path string, stats *Stats) error {
	// 1. Get Git history years
	gitBirth, gitLast, err := getGitHistoryYears(path)
	if err != nil {
		return err
	}
	colors.PrintGrey(fmt.Sprintf("[X] %-60s | Years: %s-%s", path, gitBirth, gitLast))

	// 2. Read file content
	byteContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(byteContent)

	// 3. Defensive check: Extract first year from file; use it if it's older than Git's record
	claimedBirth := extractClaimedYear(content)
	//birthYear := gitBirth
	if claimedBirth != "" && claimedBirth < gitBirth { // lexicographical comparison works for years
		panic(fmt.Sprint("	[!] Using claimed year ", claimedBirth, " over git year ", gitBirth, " for file ", path))
		//birthYear = claimedBirth
	}

	return nil
}

func getGitHistoryYears(path string) (string, string, error) {
	cmd := exec.Command("git", "log", "--follow", "--reverse", "--format=%ad", "--date=format:%Y", "--", path)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	years := strings.Fields(string(out))

	if len(years) == 0 {
		return currentYear, currentYear, nil
	}
	return years[0], years[len(years)-1], nil
}

func extractClaimedYear(content string) string {
	matches := yearRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func printSummary(s *Stats) {
	fmt.Println(strings.Repeat("-", 70))
	colors.PrintGrey(fmt.Sprintf("Total Files:    %d", s.Total))
	colors.PrintMint(fmt.Sprintf("Valid:          %d", s.Valid))
	if s.Fixed > 0 {
		colors.PrintYellow(fmt.Sprintf("Fixed:          %d", s.Fixed))
	} else if s.Valid < s.Total {
		colors.PrintRed(fmt.Sprintf("Invalid:        %d (Run with --fix to resolve)", s.Total-s.Valid))
	}
	fmt.Println(strings.Repeat("-", 70))
}
