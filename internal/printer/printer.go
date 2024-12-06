package printer

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/GraveSIN/http-probe/internal/dnsprobe"
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
			if result.ContentLength != -1 {
				parts = append(parts, fmt.Sprintf("[%s: %d]", result.ContentType, result.ContentLength))
			} else {
				parts = append(parts, fmt.Sprintf("[%s]", result.ContentType))
			}
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

func StreamDNSProbeResults(results chan dnsprobe.DNSProbeResult, outputFile string) {
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

	for result := range results {

		cyan := color.New(color.FgCyan).SprintFunc()
		blue := color.New(color.FgBlue).SprintFunc()

		output := fmt.Sprintf("%s:\n", cyan(result.Domain))

		//| MX:\n|   %v\n

		if len(result.ARecords) > 0 {
			output += fmt.Sprintf("| "+blue("A")+":\n|   %v\n", strings.Join(result.ARecords, "\n|   "))
		}
		if len(result.AAAARecords) > 0 {
			output += fmt.Sprintf("| "+blue("AAAA")+":\n|   %v\n", strings.Join(result.AAAARecords, "\n|   "))
		}
		if len(result.NSRecords) > 0 {
			output += fmt.Sprintf("| "+blue("NS")+":\n|   %v\n", strings.Join(result.NSRecords, "\n|   "))
		}
		if len(result.MXRecords) > 0 {
			output += fmt.Sprintf("| "+blue("MX")+":\n|   %v\n", strings.Join(result.MXRecords, "\n|   "))
		}
		if len(result.TXTRecords) > 0 {
			output += fmt.Sprintf("| "+blue("TXT")+":\n|   %v\n", strings.Join(result.TXTRecords, "\n|   "))
		}

		if outputFile != "" {
			fmt.Fprint(writer, output)
		} else {
			fmt.Print(output)
		}
	}
}
