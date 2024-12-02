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

		hasScheme := false
		urlToCheck := rawURL

		// METHOD 1: Character by character comparison (current implementation)
		if len(rawURL) >= 7 {
			if rawURL[0] == 'h' && rawURL[1] == 't' && rawURL[2] == 't' && rawURL[3] == 'p' {
				if rawURL[4] == ':' && rawURL[5] == '/' && rawURL[6] == '/' {
					urlToCheck = https + rawURL[7:]
					hasScheme = true
				} else if len(rawURL) >= 8 && rawURL[4] == 's' && rawURL[5] == ':' && rawURL[6] == '/' && rawURL[7] == '/' {
					hasScheme = true
				}
			}
		}

		// METHOD 2: Substring check
		// if len(rawURL) >= 5 && rawURL[0] == 'h' && rawURL[1] == 't' && rawURL[2] == 't' {
		// 	if rawURL[3:5] == "p:" {
		// 		if len(rawURL) >= 7 && rawURL[5:7] == "//" {
		// 			urlToCheck = https + rawURL[7:]
		// 			hasScheme = true
		// 		}
		// 	} else if len(rawURL) >= 6 && rawURL[3:6] == "ps:" {
		// 		if len(rawURL) >= 8 && rawURL[6:8] == "//" {
		// 			hasScheme = true
		// 		}
		// 	}
		// }

		// METHOD 3: Simplified Split check
		// if parts := strings.Split(rawURL, ":"); len(parts) > 1 {
		// 	switch parts[0] {
		// 	case "http":
		// 		urlToCheck = https + rawURL[7:]
		// 		hasScheme = true
		// 	case "https":
		// 		hasScheme = true
		// 	}
		// }

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
