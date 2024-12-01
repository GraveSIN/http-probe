package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/GraveSIN/http-probe/internal/printer"
	"github.com/GraveSIN/http-probe/internal/probe"
	"github.com/GraveSIN/http-probe/internal/utils"
	"github.com/GraveSIN/http-probe/internal/validator"
	"github.com/spf13/cobra"
)

func main() {
	var cmd = &cobra.Command{
		Use:   "http-probe",
		Short: "Probe a URL via different HTTP methods",
		Long:  "Probe a URL via different HTTP methods",
		Run:   runProbe,
	}

	cmd.Flags().StringSliceP("url", "u", []string{}, "Target URL(s) to probe")
	cmd.Flags().StringP("file", "f", "", "File containing URLs (one per line)")
	cmd.Flags().StringP("method", "X", "GET", "HTTP method to use (default: GET)")
	cmd.Flags().IntP("threads", "t", 10, "Number of concurrent threads")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().StringP("data", "d", "", "HTTP request body data")
	cmd.Flags().IntP("timeout", "T", 3, "Timeout in seconds for each request (default: 3)")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func runProbe(cmd *cobra.Command, _ []string) {
	config, err := parseConfig(cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	prober := probe.NewProber(config)
	resultsChannel := prober.Start()

	printer.StreamProbeResults(resultsChannel, config.OutputFile)
}

func parseConfig(cmd *cobra.Command) (*probe.ProberConfig, error) {
	urls, _ := cmd.Flags().GetStringSlice("url")
	urlFile, _ := cmd.Flags().GetString("file")
	method, _ := cmd.Flags().GetString("method")
	threads, _ := cmd.Flags().GetInt("threads")
	output, _ := cmd.Flags().GetString("output")
	body, _ := cmd.Flags().GetString("data")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// URLs from file
	if urlFile != "" {
		urlsFromFile, err := utils.ReadURLsFromFile(urlFile)
		if err != nil {
			return nil, err
		}
		urls = append(urls, urlsFromFile...)
	}

	// URLs from stdin
	if len(urls) == 0 && urlFile == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if url := scanner.Text(); url != "" {
					urls = append(urls, url)
				}
			}
		}
	}

	if len(urls) == 0 {
		fmt.Println("[!] at least one URL is required via -u, -f, or stdin")
		os.Exit(1)
	}

	// validate URLs and return errors
	validURLs, err := validator.ConvertDomainsToURLsAndReturnValidURLs(&urls)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(validURLs) == 0 {
		fmt.Println("[!] no valid URLs found")
		os.Exit(1)
	}

	return &probe.ProberConfig{
		URLs:       &validURLs,
		Method:     method,
		Threads:    threads,
		OutputFile: output,
		Body:       body,
		Timeout:    int(timeout),
	}, nil
}
