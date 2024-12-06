package main

import (
	"fmt"
	"os"

	"github.com/GraveSIN/http-probe/internal/dnsprobe"
	"github.com/GraveSIN/http-probe/internal/printer"
	"github.com/GraveSIN/http-probe/internal/probe"
	"github.com/spf13/cobra"
)

func main() {
	var cmd = &cobra.Command{
		Use:   "http-probe",
		Short: "Probe a URL/domain via different HTTP methods or DNS",
		Long:  "Probe a URL via different HTTP methods or DNS",
		Run:   runProbe,
	}

	cmd.Flags().StringSliceP("url", "u", []string{}, "Target URL(s) to probe")
	cmd.Flags().StringP("file", "f", "", "File containing URLs (one per line)")
	cmd.Flags().StringP("method", "X", "GET", "HTTP method to use (default: GET)")
	cmd.Flags().BoolP("dns", "", false, "Enable DNS probing instead of HTTP")
	cmd.Flags().IntP("threads", "t", 10, "Number of concurrent threads")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().StringP("data", "d", "", "HTTP request body data")
	cmd.Flags().IntP("timeout", "T", 10, "Timeout in seconds for each HTTP request or each DNS record's resolution (default: 10)")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runProbe(cmd *cobra.Command, _ []string) {

	dnsModeEnabled, _ := cmd.Flags().GetBool("dns")
	switch dnsModeEnabled {
	case true:
		// do DNS probe
		config, err := dnsprobe.ParseDNSProbeConfig(cmd)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		dnsProber := dnsprobe.NewDNSProber(config)

		resultsChannel := dnsProber.Start()

		printer.StreamDNSProbeResults(resultsChannel, config.OutputFile)

	case false:
		// do HTTP probe
		config, err := probe.ParseHTTPProbeConfig(cmd)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		prober := probe.NewProber(config)
		resultsChannel := prober.Start()

		printer.StreamProbeResults(resultsChannel, config.OutputFile)
	}

}
