package scraper

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const defaultTitle string = "untitled (unfiled)"

type OptFn func(*Options)

type Options struct {
	uri string
	doc *goquery.Document
	ctx context.Context
}

type Scraper struct {
	Options
}

func WithContext(ctx context.Context) OptFn {
	return func(o *Options) {
		o.ctx = ctx
	}
}

// Scrape fetches and parses the URL content.
func (s *Scraper) Scrape() error {
	s.doc = scrapeURL(s.uri, s.ctx)
	return nil
}

func defaults() *Options {
	return &Options{
		ctx: context.Background(),
	}
}

// Title retrieves the page title from the Scraper's Doc field, falling back
// to a default value if not found.
//
// default: `untitled (unfiled)`
func (s *Scraper) Title() string {
	t := s.doc.Find("title").Text()
	if t == "" {
		return defaultTitle
	}

	return strings.TrimSpace(t)
}

// Desc retrieves the page description from the Scraper's Doc field,
// defaulting to a predefined value if not found.
//
// default: `no description available (unfiled)`
func (s *Scraper) Desc() string {
	var desc string
	for _, selector := range []string{
		"meta[name='description']",
		"meta[name='Description']",
		"meta[property='description']",
		"meta[property='Description']",
		"meta[property='og:description']",
		"meta[property='og:Description']",
		"meta[name='og:description']",
		"meta[name='og:Description']",
	} {
		desc = s.doc.Find(selector).AttrOr("content", "")
		if desc != "" {
			break
		}
	}

	return strings.TrimSpace(desc)
}

// New creates a new Scraper.
func New(s string, opts ...OptFn) *Scraper {
	o := defaults()
	for _, opt := range opts {
		opt(o)
	}
	o.uri = s

	return &Scraper{
		Options: *o,
	}
}

// scrapeURL fetches and parses the HTML content from a URL.
func scrapeURL(s string, ctx context.Context) *goquery.Document {
	s = normalizeURL(s)
	if !isSupportedScheme(s) {
		slog.Warn("unsupported scheme", "url", s)
		return emptyDoc()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s, http.NoBody)
	if err != nil {
		slog.Error("failed to create request", "url", s, "error", err)
		return emptyDoc()
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	cl := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableCompression:  false,
		},
	}

	startTime := time.Now()
	res, err := cl.Do(req)
	if err != nil {
		d := time.Since(startTime).Milliseconds()
		slog.Error("request failed", "url", s, "error", err.Error(), "duration_ms", d, "stack", debug.Stack())
		return emptyDoc()
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			slog.Error("error closing response body", "url", s, "error", err)
		}
	}()
	slog.Info("received response", "url", s, "status", res.StatusCode, "duration", time.Since(startTime))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		logResponseStatusCode(res, s)
		return emptyDoc()
	}
	// Check content type to make sure it's HTML
	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "html") {
		slog.Warn("unexpected content type", "url", s, "content_type", contentType)
	}
	// Read the body with a limit to prevent memory issues
	// with excessively large responses
	bodyReader := io.LimitReader(res.Body, 10*1024*1024) // 10MB limit
	doc, err := goquery.NewDocumentFromReader(bodyReader)
	if err != nil {
		slog.Error("failed to parse HTML", "url", s, "error", err)
		return emptyDoc()
	}

	return doc
}

// logResponseStatusCode logs the status code of the HTTP response.
func logResponseStatusCode(res *http.Response, s string) {
	switch res.StatusCode {
	case http.StatusNotFound:
		slog.Error("page not found (404)", "url", s)
	case http.StatusForbidden, http.StatusUnauthorized:
		slog.Error("access denied", "url", s, "status_code", res.StatusCode)
	case http.StatusTooManyRequests:
		slog.Error("rate limited", "url", s)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		slog.Error("server error", "url", s, "status_code", res.StatusCode)
	}
}

func emptyDoc() *goquery.Document {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(""))
	return doc
}

func normalizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		// If no scheme, default to http
		return "http://" + raw
	}

	return raw
}

// isSupportedScheme checks if the given URL scheme is supported.
func isSupportedScheme(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)

	return scheme == "http" || scheme == "https"
}
