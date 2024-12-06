package validator

import (
	"fmt"
	urlModule "net/url"
	"strings"
)

// This will get a list of supposed URLs/domains, convert domains to URLs with an HTTPS scheme, Then validate those URLs, and return them
func ConvertDomainsToURLsAndReturnValidURLs(urlsSlice *[]string) ([]string, error) {
	urls := *urlsSlice
	validURLs := make([]string, 0, len(urls))
	const https = "https://"

	for _, rawURL := range urls {
		if len(rawURL) == 0 {
			continue
		}

		hasHttpsScheme := false
		urlToCheck := rawURL
		// Check if protocol exists but not HTTP, if yes, continue to next iteration
		if idx := strings.Index(rawURL, "://"); idx >= 0 {
			protocol := rawURL[:idx]
			switch protocol {
			case "http":
				urlToCheck = https + rawURL[idx+3:]
				hasHttpsScheme = true
			case "https":
				hasHttpsScheme = true
			default:
				continue
			}
		}

		if !hasHttpsScheme {
			urlToCheck = https + rawURL
		}

		if _, err := urlModule.ParseRequestURI(urlToCheck); err == nil {
			validURLs = append(validURLs, urlToCheck)
		} else {
			return nil, fmt.Errorf("invalid URL: %s", urlToCheck)
		}
	}

	return validURLs, nil
}
