package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func ReadURLsFromFile(urlFile string) ([]string, error) {
	var urls []string
	file, err := os.Open(urlFile)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", urlFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if url := strings.TrimSpace(scanner.Text()); url != "" {
			urls = append(urls, url)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %w", urlFile, err)
	}

	return urls, nil
}

func GetHTTPTitleFromBody(body []byte) string {
	re := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	matches := re.FindSubmatch(body)
	if len(matches) > 1 {
		return string(matches[1])
	}
	return ""
}
