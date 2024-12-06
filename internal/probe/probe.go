package probe

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GraveSIN/http-probe/internal/utils"
	"github.com/GraveSIN/http-probe/internal/validator"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

// HTTP headers
const (
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36"
	accept       = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
	acceptLang   = "en-US,en;q=0.5"
	acceptEnc    = "gzip, deflate, br"
	contentType  = "application/json"
	cacheControl = "no-cache"
	pragma       = "no-cache"
	secFetchDest = "document"
	secFetchMode = "navigate"
	secFetchSite = "none"
	secFetchUser = "?1"
	dnt          = "1"
	connection   = "close"
)

var defaultHeaders = map[string]string{
	"User-Agent":      userAgent,
	"Accept":          accept,
	"Accept-Language": acceptLang,
	"Accept-Encoding": acceptEnc,
	"Connection":      connection,
	"Content-Type":    contentType,
	"Cache-Control":   cacheControl,
	"Pragma":          pragma,
	"Sec-Fetch-Dest":  secFetchDest,
	"Sec-Fetch-Mode":  secFetchMode,
	"Sec-Fetch-Site":  secFetchSite,
	"Sec-Fetch-User":  secFetchUser,
	"DNT":             dnt,
}

// ProbeResult represents the result of an HTTP probe containing various response details
type ProbeResult struct {
	URL              string
	StatusLine       string
	ServerHeader     string
	RedirectLocation string
	Title            string
	ContentType      string
	ContentLength    int
	PoweredByHeader  string
	TimeTaken        time.Duration
}

// ProberConfig contains the configuration options for the HTTP prober
type ProberConfig struct {
	URLs       *[]string
	Threads    int
	Timeout    int
	Method     string
	OutputFile string
	Body       string
	DNSMode    bool
}

// Prober handles the HTTP probing operations
type Prober struct {
	config    *ProberConfig
	client    *fasthttp.Client
	results   chan ProbeResult
	workPool  chan string
	waitGroup sync.WaitGroup
}

func ParseHTTPProbeConfig(cmd *cobra.Command) (*ProberConfig, error) {
	urls, _ := cmd.Flags().GetStringSlice("url")
	urlFile, _ := cmd.Flags().GetString("file")
	method, _ := cmd.Flags().GetString("method")
	threads, _ := cmd.Flags().GetInt("threads")
	output, _ := cmd.Flags().GetString("output")
	body, _ := cmd.Flags().GetString("data")
	timeout, _ := cmd.Flags().GetInt("timeout")
	dnsMode, _ := cmd.Flags().GetBool("dns")

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

	return &ProberConfig{
		URLs:       &validURLs,
		Method:     method,
		Threads:    threads,
		OutputFile: output,
		Body:       body,
		DNSMode:    dnsMode,
		Timeout:    int(timeout),
	}, nil
}

// NewProber initializes a new Prober instance with the provided configuration
func NewProber(config *ProberConfig) *Prober {
	// Pre-calculate buffer sizes based on URL count
	urlCount := len(*config.URLs)
	bufferSize := config.Threads * 2

	return &Prober{
		config:   config,
		client:   createOptimizedClient(config),
		results:  make(chan ProbeResult, bufferSize),
		workPool: make(chan string, urlCount),
	}
}

// createOptimizedClient creates a fasthttp.Client with optimized settings for HTTP probing
func createOptimizedClient(config *ProberConfig) *fasthttp.Client {
	dialer := &fasthttp.TCPDialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp", "1.1.1.1:53")
			},
		},
	}

	timeout := time.Duration(config.Timeout) * time.Second
	return &fasthttp.Client{
		MaxConnsPerHost:               config.Threads * 2,
		ReadTimeout:                   timeout,
		MaxIdleConnDuration:           timeout,
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
		MaxConnWaitTimeout:            timeout,
		MaxConnDuration:               time.Minute,
		MaxIdemponentCallAttempts:     1,
		MaxResponseBodySize:           10 * 1024 * 1024,
		TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
		Dial:                          dialer.Dial,
	}
}

// Start begins the probing process by initializing workers and returning a channel for results
func (p *Prober) Start() chan ProbeResult {
	p.waitGroup.Add(p.config.Threads)

	go p.initializeWorkPool()

	for range p.config.Threads {
		go p.worker()
	}

	go p.waitAndClose()

	return p.results
}

// initializeWorkPool populates the work pool with URLs to be processed
func (p *Prober) initializeWorkPool() {
	for _, url := range *p.config.URLs {
		p.workPool <- url
	}
	close(p.workPool)
}

// worker processes URLs from the work pool until the pool is empty
func (p *Prober) worker() {
	defer p.waitGroup.Done()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	for url := range p.workPool {
		result := p.probeURL(url, req, resp)
		p.results <- result
	}
}

// probeURL performs an HTTP request to the specified URL and returns the probe results
func (p *Prober) probeURL(url string, req *fasthttp.Request, resp *fasthttp.Response) ProbeResult {
	req.Reset()
	resp.Reset()

	req.SetRequestURI(url)
	req.Header.SetMethod(p.config.Method)

	for key, value := range defaultHeaders {
		req.Header.Set(key, value)
	}

	if p.config.Body != "" {
		req.SetBodyString(p.config.Body)
		bodyLen := len(p.config.Body)
		req.Header.SetContentLength(bodyLen)

		if bodyLen > 0 {
			req.Header.Set("Content-Type", detectContentType(p.config.Body[0], p.config.Body))
		}
	}

	startTime := time.Now()
	if err := p.makeRequest(req, resp, url); err != nil {
		return ProbeResult{}
	}

	return createProbeResult(url, resp, startTime, p)
}

// detectContentType determines the appropriate Content-Type header based on the request body
// It supports JSON, XML, form-urlencoded, and plain text detection
func detectContentType(firstByte byte, body string) string {
	switch {
	case firstByte == '{' || firstByte == '[':
		return "application/json"
	case firstByte == '<':
		return "application/xml"
	case strings.Contains(body, "="):
		return "application/x-www-form-urlencoded"
	default:
		return "text/plain"
	}
}

// makeRequest performs the HTTP request with fallback to HTTP if HTTPS fails
func (p *Prober) makeRequest(req *fasthttp.Request, resp *fasthttp.Response, url string) error {
	if err := p.client.DoTimeout(req, resp, time.Duration(p.config.Timeout)*time.Second); err != nil {
		if strings.HasPrefix(url, "https://") {
			url = "http://" + url[8:]
			req.SetRequestURI(url)
			return p.client.DoTimeout(req, resp, time.Duration(p.config.Timeout))
		}
		return err
	}
	return nil
}

// createProbeResult constructs a ProbeResult struct from the HTTP response
// It extracts information from the response
func createProbeResult(url string, resp *fasthttp.Response, startTime time.Time, p *Prober) ProbeResult {

	contentType := strings.Split(string(resp.Header.Peek("Content-Type")), ";")[0]
	contentLength := resp.Header.ContentLength()

	if contentLength == -1 {
		contentLength = len(resp.Body())
	}

	return ProbeResult{
		URL:        url,
		StatusLine: fmt.Sprintf("%d %s", resp.StatusCode(), fasthttp.StatusMessage(resp.StatusCode())),

		ServerHeader:     string(resp.Header.Peek("Server")),
		ContentType:      contentType,
		RedirectLocation: string(resp.Header.Peek("Location")),
		Title:            utils.GetHTTPTitleFromBody(resp.Body()),
		ContentLength:    contentLength,
		PoweredByHeader:  string(resp.Header.Peek("X-Powered-By")),
		TimeTaken:        time.Since(startTime),
	}
}

// waitAndClose waits for all workers to complete their tasks and closes the results channel
func (p *Prober) waitAndClose() {
	p.waitGroup.Wait()
	close(p.results)
}
