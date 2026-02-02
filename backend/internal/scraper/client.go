package scraper

import (
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Client is an HTTP client for scraping
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a new scraper client
func NewClient(userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		userAgent: userAgent,
	}
}

// Fetch fetches a URL and returns the HTML content
func (c *Client) Fetch(url string) (string, error) {
	return c.FetchWithRetry(url, 2)
}

// FetchWithRetry fetches a URL with retry logic
func (c *Client) FetchWithRetry(url string, maxRetries int) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("Connection", "keep-alive")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch URL: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
			var reader io.Reader = resp.Body

			// Handle gzip decompression
			if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
				gzipReader, err := gzip.NewReader(resp.Body)
				if err != nil {
					lastErr = fmt.Errorf("failed to create gzip reader: %w", err)
					continue
				}
				reader = gzipReader
			}

			content, err := io.ReadAll(reader)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				continue
			}

			return string(content), nil
		}

		// For non-200 status codes, don't retry
		lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		break
	}

	return "", lastErr
}

// FetchDetail fetches a product detail page with longer timeout and retry
func (c *Client) FetchDetail(url string) (string, error) {
	// Create a client with longer timeout for detail pages
	detailClient := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     60 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")

	resp, err := detailClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body

	// Handle gzip decompression
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		reader = gzipReader
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(content), nil
}

// ExtractText extracts text content from HTML, removing tags
func ExtractText(html string) string {
	// Remove script and style tags
	re := regexp.MustCompile(`<(script|style)[^>]*>.*?</\1>`)
	html = re.ReplaceAllString(html, "")

	// Remove HTML tags
	re = regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, " ")

	// Clean up whitespace
	text = strings.Join(strings.Fields(text), " ")

	return text
}

// CleanPrice extracts numeric price from string
func CleanPrice(priceStr string) float64 {
	// Remove currency symbols and extract numbers
	re := regexp.MustCompile(`[0-9,]+\.?[0-9]*`)
	matches := re.FindAllString(priceStr, -1)

	if len(matches) == 0 {
		return 0
	}

	// Get the last match (usually the actual price)
	priceStr = matches[len(matches)-1]

	// Remove commas
	priceStr = strings.ReplaceAll(priceStr, ",", "")

	var price float64
	if _, err := fmt.Sscanf(priceStr, "%f", &price); err != nil {
		return 0
	}

	return price
}

// NormalizeCategory normalizes category names
func NormalizeCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))

	categoryMap := map[string]string{
		"macbook air":    "Mac",
		"macbook pro":    "Mac",
		"mac mini":       "Mac",
		"mac studio":     "Mac",
		"imac":           "Mac",
		"mac":            "Mac",
		"ipad pro":       "iPad",
		"ipad air":       "iPad",
		"ipad mini":      "iPad",
		"ipad":           "iPad",
		"iphone":         "iPhone",
		"apple watch":    "Watch",
		"watch":          "Watch",
		"airpods":        "Accessory",
		"homepod":        "Accessory",
		"appletv":        "Accessory",
		"pencil":         "Accessory",
		"magic keyboard": "Accessory",
		"magic mouse":    "Accessory",
		"trackpad":       "Accessory",
		"display":        "Accessory",
		"accessories":    "Accessory",
		"accessory":      "Accessory",
	}

	for key, val := range categoryMap {
		if strings.Contains(category, key) {
			return val
		}
	}

	return category
}

// ParseSpecs extracts specs from product name
func ParseSpecs(name, specs string) string {
	parts := []string{}

	// Add name if it contains useful info
	if specs == "" {
		specs = name
	}

	// Clean up specs
	specs = strings.TrimSpace(specs)
	specs = strings.ReplaceAll(specs, "  ", " ")

	if specs != "" {
		parts = append(parts, specs)
	}

	return strings.Join(parts, " | ")
}
