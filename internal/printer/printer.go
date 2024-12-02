package printer

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/GraveSIN/http-probe/internal/probe"
	"github.com/fatih/color"
)

func StreamProbeResults(results chan probe.ProbeResult, outputFile string) {
	var f *os.File
	var writer *bufio.Writer

	if outputFile != "" {
		var err error
		f, err = os.Create(outputFile)
		if err != nil {
			log.Fatalf("[+] Failed to create output file: %v", err)
		}
		defer f.Close()
		writer = bufio.NewWriter(f)
		defer writer.Flush()
	}

	redStatus := color.New(color.FgRed).SprintFunc()
	yellowStatus := color.New(color.FgYellow).SprintFunc()
	greenStatus := color.New(color.FgGreen).SprintFunc()

	for result := range results {
		if result.StatusLine == "" {
			continue
		}
		var coloredStatus string
		switch result.StatusLine[0] {
		case '1':
			coloredStatus = greenStatus(result.StatusLine)
		case '2':
			coloredStatus = greenStatus(result.StatusLine)
		case '3':
			coloredStatus = yellowStatus(result.StatusLine)
		case '4':
			coloredStatus = redStatus(result.StatusLine)
		case '5':
			coloredStatus = yellowStatus(result.StatusLine)
		default:
			coloredStatus = redStatus(result.StatusLine)
		}

		// Build output parts dynamically
		parts := []string{
			fmt.Sprintf("[+] %s: [%s]", result.URL, coloredStatus),
		}

		if result.RedirectLocation != "" {
			parts = append(parts, "->", result.RedirectLocation)
		}

		result.Title = strings.TrimSpace(result.Title)
		if result.Title != "" {
			parts = append(parts, "["+result.Title+"]")
		}

		if result.ContentType != "" {
			if result.ContentLength != 0 {
				parts = append(parts, fmt.Sprintf("[%s: %d]", result.ContentType, result.ContentLength))
			} else {
				parts = append(parts, fmt.Sprintf("[%s]", result.ContentType))
			}
		}

		if len(result.SupportedMethods) > 0 {
			parts = append(parts, fmt.Sprintf("%v", result.SupportedMethods))
		}
		if result.ServerHeader != "" {
			parts = append(parts, result.ServerHeader)
		}
		if result.PoweredByHeader != "" {
			parts = append(parts, result.PoweredByHeader)
		}
		// time duration in ms
		parts = append(parts, fmt.Sprintf("%dms", result.TimeTaken.Milliseconds()))

		output := fmt.Sprintf("%s\n", strings.Join(parts, " "))

		if outputFile != "" {
			fmt.Fprint(writer, output)
		} else {
			fmt.Print(output)
		}
	}
}
