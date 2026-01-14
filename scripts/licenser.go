// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/offchainlabs/nitro/util/colors"
)

const (
	company    = "Offchain Labs, Inc."
	licenseURL = "https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md"
)

var (
	currentYear         = strconv.Itoa(time.Now().Year())
	supportedExtensions = []string{".go", ".rs"}
	yearRegex           = regexp.MustCompile(`Copyright\s+(\d{4})(?:-(\d{4}))?`)
	fixFlag             = flag.Bool("fix", false, "Update files with the correct license header")
)

type stats struct {
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

	stats := &stats{Total: len(files)}
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

func processFile(path string, stats *stats) error {
	// 1. Get Git history years
	gitBirth, lastYear, err := getGitHistoryYears(path)
	if err != nil {
		return err
	}

	// 2. Read file content
	byteContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(byteContent)

	// 3. Defensive check: Extract first year from file; use it if it's older than Git's record
	claimedBirth := extractClaimedYear(content)
	birthYear := gitBirth
	if claimedBirth != "" && claimedBirth < gitBirth { // lexicographical comparison works for years
		colors.PrintGrey("[!] Using claimed year ", claimedBirth, " over git year ", gitBirth, " for file ", path)
		birthYear = claimedBirth
	}

	// 4. Validate existing header
	switch validateHeader(content, birthYear, lastYear) {
	case validHeader:
		colors.PrintGrey(fmt.Sprintf("[✓] %-60s | License years: %s-%s", path, birthYear, lastYear))
		stats.Valid++
		return nil
	case missingURL:
		colors.PrintRed(fmt.Sprintf("[X] %-60s | Reason: Missing license URL", path))
	case missingCopyright:
		colors.PrintRed(fmt.Sprintf("[X] %-60s | Reason: Missing copyright line", path))
	case incorrectYears:
		colors.PrintRed(fmt.Sprintf("[X] %-60s | Reason: Incorrect year range (Expected %s-%s)", path, birthYear, lastYear))
	}

	// 5. Handle inconsistency
	if *fixFlag {
		if err := applyFix(path, content, birthYear); err != nil {
			return fmt.Errorf("failed to apply fix: %w", err)
		}
		colors.PrintYellow(fmt.Sprintf("[+] %-60s | Set license to %s-%s range", path, birthYear, currentYear))
		stats.Fixed++
	}
	return nil
}

func getGitHistoryYears(path string) (created, lastModified string, err error) {
	cmd := exec.Command("git", "log", "--follow", "--format=%ad", "--date=format:%Y", "--", path)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	years := strings.Fields(string(out))

	if len(years) == 0 {
		return currentYear, currentYear, nil
	}
	return years[len(years)-1], years[0], nil
}

func extractClaimedYear(content string) string {
	matches := yearRegex.FindStringSubmatch(content)
	// matches[0] is the full match, matches[1] is the first capturing group (the first year)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

type validationResult int

const (
	validHeader validationResult = iota
	missingCopyright
	missingURL
	incorrectYears
)

func validateHeader(content string, birth, update string) validationResult {
	// Only check the first 500 characters to ensure the header is at the top
	searchArea := content
	if len(content) > 500 {
		searchArea = content[:500]
	}

	if !strings.Contains(strings.ToLower(searchArea), strings.ToLower(licenseURL)) {
		return missingURL
	}
	if match := yearRegex.FindStringSubmatch(searchArea); match == nil {
		return missingCopyright
	}
	expectedCopyright := fmt.Sprintf("Copyright %s-%s, %s", birth, update, company)
	if !strings.Contains(searchArea, expectedCopyright) {
		return incorrectYears
	}
	return validHeader
}

func applyFix(path, content, birthYear string) error {
	header := fmt.Sprintf("// Copyright %s-%s, %s\n// For license information, see %s\n",
		birthYear, currentYear, company, licenseURL)

	lines := strings.Split(content, "\n")
	startIdx := 0
	// Skip existing copyright/comment block to avoid duplicates
	if len(lines) > 0 && strings.HasPrefix(lines[0], "// Copyright") {
		for startIdx < len(lines) && (strings.HasPrefix(lines[startIdx], "//")) {
			startIdx++
		}
	}

	newContent := header + strings.Join(lines[startIdx:], "\n")
	return os.WriteFile(path, []byte(newContent), 0644) // #nosec G306
}

func printSummary(s *stats) {
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
