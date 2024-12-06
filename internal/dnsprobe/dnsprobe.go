package dnsprobe

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/GraveSIN/http-probe/internal/utils"
	"github.com/spf13/cobra"
	"net"
)

type DNSProbeConfig struct {
	Domains    *[]string
	Threads    int
	OutputFile string
	Timeout    int
}

type DNSProbeResult struct {
	Domain      string
	TXTRecords  []string
	NSRecords   []string
	ARecords    []string
	AAAARecords []string
	MXRecords   []string
}

type DNSProber struct {
	config    *DNSProbeConfig
	results   chan DNSProbeResult
	workPool  chan string
	waitGroup sync.WaitGroup
	resolver  *net.Resolver
}

func ParseDNSProbeConfig(cmd *cobra.Command) (*DNSProbeConfig, error) {
	domains, _ := cmd.Flags().GetStringSlice("url")
	domainFile, _ := cmd.Flags().GetString("file")
	threads, _ := cmd.Flags().GetInt("threads")
	timeout, _ := cmd.Flags().GetInt("timeout")
	outputFile, _ := cmd.Flags().GetString("output")

	// Domains from file
	if domainFile != "" {
		domainsFromFile, err := utils.ReadURLsFromFile(domainFile)
		if err != nil {
			return nil, err
		}
		domains = append(domains, domainsFromFile...)
	}

	// Domains from stdin
	if len(domains) == 0 && domainFile == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if domain := scanner.Text(); domain != "" {
					domains = append(domains, domain)
				}
			}
		}
	}

	if len(domains) == 0 {
		fmt.Println("[!] at least one domain is required via -u, -f, or stdin")
		os.Exit(1)
	}

	return &DNSProbeConfig{
		Domains:    &domains,
		Threads:    threads,
		OutputFile: outputFile,
		Timeout: timeout,
	}, nil
}

func NewDNSProber(config *DNSProbeConfig) *DNSProber {
	resolver := &net.Resolver{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Duration(config.Timeout) * time.Second,
			}
			return d.DialContext(ctx, "udp", "1.1.1.1:53")
		},
	}

	return &DNSProber{
		config:   config,
		resolver: resolver,
		results:  make(chan DNSProbeResult, config.Threads*2),
		workPool: make(chan string, len(*config.Domains)),
	}
}

func (p *DNSProber) Start() chan DNSProbeResult {
	p.waitGroup.Add(p.config.Threads)

	go p.initializeWorkPool()

	for range p.config.Threads {
		go p.worker()
	}

	go p.waitAndClose()

	return p.results
}

func (p *DNSProber) initializeWorkPool() {
	for _, domain := range *p.config.Domains {
		p.workPool <- domain
	}
	close(p.workPool)
}

func (p *DNSProber) worker() {
	defer p.waitGroup.Done()

	for domain := range p.workPool {
		result := p.dnsProbeDomain(domain)
		p.results <- result
	}
}

func (p *DNSProber) waitAndClose() {
	p.waitGroup.Wait()
	close(p.results)
}

func (p *DNSProber) dnsProbeDomain(domain string) DNSProbeResult {
	if err := utils.ValidateDomain(domain); err != nil {
		return DNSProbeResult{}
	}

	result := DNSProbeResult{
		Domain: domain,
	}

	txtCtx, txtCancel := context.WithTimeout(context.Background(), time.Duration(p.config.Timeout)*time.Second)
	defer txtCancel()
	if txtRecords, err := p.resolver.LookupTXT(txtCtx, domain); err == nil {
		result.TXTRecords = txtRecords
	}

	nsCtx, nsCancel := context.WithTimeout(context.Background(), time.Duration(p.config.Timeout)*time.Second)
	defer nsCancel()
	if nsRecords, err := p.resolver.LookupNS(nsCtx, domain); err == nil && len(nsRecords) > 0 {
		ns := make([]string, len(nsRecords))
		for i, record := range nsRecords {
			ns[i] = record.Host
		}
		result.NSRecords = ns
	}

	ipCtx, ipCancel := context.WithTimeout(context.Background(), time.Duration(p.config.Timeout)*time.Second)
	defer ipCancel()
	if aRecords, err := p.resolver.LookupIPAddr(ipCtx, domain); err == nil {
		var ip4, ip6 []string
		for _, ip := range aRecords {
			if ip.IP.To4() != nil {
				 ip4 = append(ip4, ip.IP.String())
			} else {
				ip6 = append(ip6, ip.IP.String())
			}
		}
		result.ARecords = ip4
		result.AAAARecords = ip6
	}

	mxCtx, mxCancel := context.WithTimeout(context.Background(), time.Duration(p.config.Timeout)*time.Second)
	defer mxCancel()
	if mxRecords, err := p.resolver.LookupMX(mxCtx, domain); err == nil && len(mxRecords) > 0 {
		mx := make([]string, len(mxRecords))
		for i, record := range mxRecords {
			mx[i] = record.Host
		}
		result.MXRecords = mx
	}

	return result
}
