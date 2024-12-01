package validator

import (
	"fmt"
	urlModule "net/url"
)

func ConvertDomainsToURLsAndReturnValidURLs(urlsSlice *[]string) ([]string, error) {
	urls := *urlsSlice
	validURLs := make([]string, 0, len(urls))

	const https = "https://"

	for _, rawURL := range urls {
		if len(rawURL) == 0 {
			continue
		}

		// Fast scheme check using length and direct character comparison
		hasScheme := false
		if len(rawURL) >= 7 { // length of "http://"
			if rawURL[0] == 'h' && rawURL[1] == 't' && rawURL[2] == 't' && rawURL[3] == 'p' {
				if rawURL[4] == ':' && rawURL[5] == '/' && rawURL[6] == '/' {
					hasScheme = true
				} else if len(rawURL) >= 8 && rawURL[4] == 's' && rawURL[5] == ':' && rawURL[6] == '/' && rawURL[7] == '/' {
					hasScheme = true
				}
			}
		}

		urlToCheck := rawURL
		if !hasScheme {
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
