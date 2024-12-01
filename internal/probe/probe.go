package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/GraveSIN/http-probe/internal/utils"
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
	SupportedMethods []string
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
}

// Prober handles the HTTP probing operations
type Prober struct {
	config    *ProberConfig
	client    *fasthttp.Client
	results   chan ProbeResult
	workPool  chan string
	waitGroup sync.WaitGroup
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
				d := net.Dialer{
					Timeout: time.Duration(config.Timeout) * time.Second,
				}
				return d.DialContext(ctx, "udp", "1.1.1.1:53") // Cloudflare DNS
			},
		},
	}

	return &fasthttp.Client{
		MaxConnsPerHost:               config.Threads * 2,
		ReadTimeout:                   time.Duration(config.Timeout) * time.Second,
		WriteTimeout:                  time.Duration(config.Timeout) * time.Second,
		MaxIdleConnDuration:           time.Second,
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
		MaxConnWaitTimeout:            time.Second,
		MaxConnDuration:               time.Minute,
		MaxIdemponentCallAttempts:     1,
		MaxResponseBodySize:           10 * 1024 * 1024, // 10MB limit
		TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
		Dial:                          dialer.Dial,
	}
}

// Start begins the probing process by initializing workers and returning a channel for results
func (p *Prober) Start() chan ProbeResult {
	go p.initializeWorkPool()

	for i := 0; i < p.config.Threads; i++ {
		p.waitGroup.Add(1)
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
	if err := p.client.Do(req, resp); err != nil {
		if strings.HasPrefix(url, "https://") {
			url = "http://" + strings.TrimPrefix(url, "https://")
			req.SetRequestURI(url)
			return p.client.Do(req, resp)
		}
		return err
	}
	return nil
}

// createProbeResult constructs a ProbeResult struct from the HTTP response
// It extracts information from the response
func createProbeResult(url string, resp *fasthttp.Response, startTime time.Time, p *Prober) ProbeResult {
	return ProbeResult{
		URL:        url,
		StatusLine: fmt.Sprintf("%d %s", resp.StatusCode(), fasthttp.StatusMessage(resp.StatusCode())),

		ServerHeader:     string(resp.Header.Peek("Server")),
		ContentType:      string(resp.Header.Peek("Content-Type")),
		RedirectLocation: string(resp.Header.Peek("Location")),
		Title:            utils.GetHTTPTitleFromBody(resp.Body()),
		SupportedMethods: p.determineSupportedMethods(url),
		ContentLength:    resp.Header.ContentLength(),
		PoweredByHeader:  string(resp.Header.Peek("X-Powered-By")),
		TimeTaken:        time.Since(startTime),
	}
}

// waitAndClose waits for all workers to complete their tasks and closes the results channel
func (p *Prober) waitAndClose() {
	p.waitGroup.Wait()
	close(p.results)
}

// determineSupportedMethods sends an OPTIONS request to discover supported HTTP methods
func (p *Prober) determineSupportedMethods(url string) []string {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod("OPTIONS")

	if err := p.client.Do(req, resp); err != nil {
		return nil
	}

	allow := string(resp.Header.Peek("Allow"))
	if allow == "" {
		return nil
	}

	methods := strings.Split(allow, ",")
	for i := range methods {
		methods[i] = strings.TrimSpace(methods[i])
	}
	return methods
}
