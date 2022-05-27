// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"time"
)

const initialMaxRecurseDepth uint16 = 8

// RestfulServerURLsFromList reads a list of Restful server URLs from a remote URL.
// The contents at the remote URL are parsed into a series of whitespace-separated words.
// Each word is interpreted as the URL of a Restful server, except that if a word is "LIST"
//    (case-insensitive) then the following word is interpreted as the URL of another list,
//    which is recursively fetched. The depth of recursion is limited to initialMaxRecurseDepth.
func RestfulServerURLsFromList(ctx context.Context, listUrl string) ([]string, error) {
	client := &http.Client{}
	urls, err := restfulServerURLsFromList(ctx, client, listUrl, initialMaxRecurseDepth)
	if err != nil {
		return nil, err
	}

	// deduplicate the list of URL strings
	seen := make(map[string]bool)
	dedupedUrls := []string{}
	for _, url := range urls {
		if !seen[url] {
			seen[url] = true
			dedupedUrls = append(dedupedUrls, url)
		}
	}

	return dedupedUrls, nil
}

func restfulServerURLsFromList(ctx context.Context, client *http.Client, listUrl string, maxRecurseDepth uint16) ([]string, error) {
	urls := []string{}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, listUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		word := scanner.Text()
		if strings.ToLower(word) == "list" {
			if maxRecurseDepth > 0 && scanner.Scan() {
				word = scanner.Text()
				subUrls, err := restfulServerURLsFromList(ctx, client, word, maxRecurseDepth-1)
				if err != nil {
					urls = append(urls, subUrls...)
				}
			}
		} else {
			urls = append(urls, word)
		}
	}
	return urls, nil
}

const maxListFetchTime = time.Minute

func StartRestfulServerListFetchDaemon(ctx context.Context, listUrl string, updatePeriod time.Duration) <-chan []string {
	updateChan := make(chan []string)

	downloadAndSend := func() bool { // download and send once, return true iff success
		subCtx, subCtxCancel := context.WithTimeout(ctx, maxListFetchTime)
		defer subCtxCancel()

		urls, err := RestfulServerURLsFromList(subCtx, listUrl)
		if err != nil {
			return false
		}
		select {
		case updateChan <- urls:
			return true
		case <-ctx.Done():
			return false
		}
	}

	go func() {
		defer close(updateChan)

		// send the first result immediately
		if !downloadAndSend() {
			return
		}

		// now send periodically
		ticker := time.NewTicker(updatePeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !downloadAndSend() {
					return
				}
			}
		}
	}()

	return updateChan
}
