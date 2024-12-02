package validator

import (
	"testing"
)

func BenchmarkURLValidation(b *testing.B) {
	// RUN: go test -bench=. -benchmem, in same directory
	// Create a mix of URLs with different schemes
	urls := []string{
		"http://example.com",
		"https://secure.example.com",
		"example.com",
		"http://very-long-domain-name-with-many-subdomains.example.com",
		"https://another-secure-domain.example.com",
		"domain-without-scheme.com",
		"http://domain-with-http.com",
		"https://domain-with-https.com",
		// Add more URLs to make the test more realistic
	}

	// Create a larger dataset by repeating the URLs
	largeURLs := make([]string, 0, len(urls)*5000000)
	for i := 0; i < 5000000; i++ {
		largeURLs = append(largeURLs, urls...)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testURLs := make([]string, len(largeURLs))
		copy(testURLs, largeURLs)
		_, _ = ConvertDomainsToURLsAndReturnValidURLs(&testURLs)
	}
}

func TestURLValidation(t *testing.T) {
	testCases := []struct {
		name     string
		inputs   []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Valid URLs with mixed schemes",
			inputs:   []string{"http://example.com", "https://secure.com", "plain.com"},
			expected: []string{"https://example.com", "https://secure.com", "https://plain.com"},
			wantErr:  false,
		},
		{
			name:     "Empty URL in list",
			inputs:   []string{"http://example.com", "", "plain.com"},
			expected: []string{"https://example.com", "https://plain.com"},
			wantErr:  false,
		},
		{
			name:    "Invalid URL",
			inputs:  []string{"http://example.com", "not a url", "plain.com"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputs := make([]string, len(tc.inputs))
			copy(inputs, tc.inputs)

			results, err := ConvertDomainsToURLsAndReturnValidURLs(&inputs)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(results) != len(tc.expected) {
				t.Errorf("expected %d results, got %d", len(tc.expected), len(results))
				return
			}

			for i, result := range results {
				if result != tc.expected[i] {
					t.Errorf("expected %s, got %s at index %d", tc.expected[i], result, i)
				}
			}
		})
	}
}
