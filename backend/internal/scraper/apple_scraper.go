package scraper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"apple-price/internal/model"
)

const (
	cnBaseURL = "https://www.apple.com.cn/shop/refurbished"
)

// AppleScraper scrapes Apple's refurbished product pages
type AppleScraper struct {
	client *Client
}

// NewAppleScraper creates a new Apple scraper instance
func NewAppleScraper(client *Client) *AppleScraper {
	return &AppleScraper{
		client: client,
	}
}

// ScrapeAll scrapes all products from China region
func (s *AppleScraper) ScrapeAll() ([]*model.Product, error) {
	return s.ScrapeRegion("cn", cnBaseURL)
}

// ScrapeRegion scrapes products from a specific region
func (s *AppleScraper) ScrapeRegion(region, baseURL string) ([]*model.Product, error) {
	// Category pages to scrape
	// Note: iPhone is not available as refurbished in China/HK
	// Apple TV is only available in Hong Kong, but we'll skip it for now
	categoryPages := map[string]string{
		"Mac":       baseURL + "/mac",
		"iPad":      baseURL + "/ipad",
		// "iPhone":  baseURL + "/iphone", // Not available in China/HK
		"Watch":     baseURL + "/watch",
		"AirPods":   baseURL + "/airpods",
		"HomePod":   baseURL + "/homepod",
		// "Apple TV":  baseURL + "/appletv", // Only in HK, skip for now
		"Accessories": baseURL + "/accessories",
	}

	var allProducts []*model.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Scrape each category (detail scraping is now async via DetailScraper)
	for category, catURL := range categoryPages {
		wg.Add(1)
		go func(cat, url string) {
			defer wg.Done()

			products, err := s.scrapeCategoryPage(cat, region, url)
			if err != nil {
				fmt.Printf("Error scraping %s: %v\n", cat, err)
				return
			}

			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}(category, catURL)
	}

	wg.Wait()

	return allProducts, nil
}

// scrapeCategoryPage scrapes a single category page
func (s *AppleScraper) scrapeCategoryPage(category, region, url string) ([]*model.Product, error) {
	html, err := s.client.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	// Extract the REFURB_GRID_BOOTSTRAP JSON data
	bootstrapData, err := s.extractBootstrapData(html)
	if err != nil {
		return nil, fmt.Errorf("failed to extract bootstrap data: %w", err)
	}

	// Parse the tiles (products)
	products := s.parseTilesFromBootstrap(bootstrapData, category, region, url)

	return products, nil
}

// extractBootstrapData extracts the window.REFURB_GRID_BOOTSTRAP JSON data
func (s *AppleScraper) extractBootstrapData(html string) (map[string]interface{}, error) {
	// Find the start of REFURB_GRID_BOOTSTRAP variable
	startIdx := strings.Index(html, "window.REFURB_GRID_BOOTSTRAP")
	if startIdx == -1 {
		return nil, fmt.Errorf("REFURB_GRID_BOOTSTRAP not found")
	}

	// Find the opening brace after the equals sign
	equalsIdx := strings.Index(html[startIdx:], "=")
	if equalsIdx == -1 {
		return nil, fmt.Errorf("invalid REFURB_GRID_BOOTSTRAP format")
	}
	startIdx += equalsIdx + 1

	// Skip whitespace
	for startIdx < len(html) {
		c := html[startIdx]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		startIdx++
	}

	if startIdx >= len(html) || html[startIdx] != '{' {
		return nil, fmt.Errorf("expected opening brace, got: %v", html[startIdx:startIdx+10])
	}

	// Find the matching closing brace by counting braces
	braceCount := 0
	i := startIdx
	maxLen := len(html)

	for i < maxLen {
		c := html[i]
		if c == '{' {
			braceCount++
		} else if c == '}' {
			braceCount--
			if braceCount == 0 {
				// Found the matching closing brace
				break
			}
		}
		i++
	}

	if braceCount != 0 {
		return nil, fmt.Errorf("unbalanced braces in JSON (count: %d)", braceCount)
	}

	jsonStr := html[startIdx : i+1]

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse bootstrap JSON: %w, json length: %d", err, len(jsonStr))
	}

	return data, nil
}

// parseTilesFromBootstrap parses product tiles from bootstrap data
func (s *AppleScraper) parseTilesFromBootstrap(bootstrap map[string]interface{}, category, region, pageURL string) []*model.Product {
	products := []*model.Product{}

	tiles, ok := bootstrap["tiles"].([]interface{})
	if !ok {
		return products
	}

	now := time.Now()

	for _, tileInterface := range tiles {
		tile, ok := tileInterface.(map[string]interface{})
		if !ok {
			continue
		}

		product := s.parseTile(tile, category, region, pageURL, now)
		if product != nil {
			products = append(products, product)
		}
	}

	return products
}

// parseTile parses a single product tile
func (s *AppleScraper) parseTile(tile map[string]interface{}, category, region, pageURL string, timestamp time.Time) *model.Product {
	// Extract title
	title, _ := tile["title"].(string)
	if title == "" {
		return nil
	}

	// Extract price data
	price := 0.0
	originalPrice := 0.0

	if priceObj, ok := tile["price"].(map[string]interface{}); ok {
		// currentPrice can be a string, float64, or map[string]interface{}
		if currentPrice, ok := priceObj["currentPrice"].(string); ok {
			price = CleanPrice(currentPrice)
		} else if currentPrice, ok := priceObj["currentPrice"].(float64); ok {
			price = currentPrice
		} else if currentPriceMap, ok := priceObj["currentPrice"].(map[string]interface{}); ok {
			// Apple China uses: {"amount": "RMB 3,799", "raw_amount": "3799.00"}
			if rawAmount, ok := currentPriceMap["raw_amount"].(string); ok {
				price = CleanPrice(rawAmount)
			} else if amount, ok := currentPriceMap["amount"].(string); ok {
				price = CleanPrice(amount)
			}
		}

		// originalPrice can be a string, float64, or map[string]interface{}
		if origPrice, ok := priceObj["originalPrice"].(string); ok {
			originalPrice = CleanPrice(origPrice)
		} else if origPrice, ok := priceObj["originalPrice"].(float64); ok {
			originalPrice = origPrice
		} else if origPriceMap, ok := priceObj["originalPrice"].(map[string]interface{}); ok {
			if rawAmount, ok := origPriceMap["raw_amount"].(string); ok {
				originalPrice = CleanPrice(rawAmount)
			} else if amount, ok := origPriceMap["amount"].(string); ok {
				originalPrice = CleanPrice(amount)
			}
		}

		// If no original price, estimate it (typically 15% higher for refurbished)
		if originalPrice == 0 && price > 0 {
			originalPrice = price / 0.85
		}
	}

	// Calculate discount
	discount := 0.0
	if originalPrice > 0 && price > 0 {
		discount = (1 - price/originalPrice) * 100
	}

	// Extract product URL
	productURL := ""
	if detailsURL, ok := tile["productDetailsUrl"].(string); ok {
		if strings.HasPrefix(detailsURL, "/") {
			if region == "cn" {
				productURL = "https://www.apple.com.cn" + detailsURL
			} else {
				productURL = "https://www.apple.com.hk" + detailsURL
			}
		} else {
			productURL = detailsURL
		}
	}

	// Extract image URL
	imageURL := ""
	if imageObj, ok := tile["image"].(map[string]interface{}); ok {
		if sources, ok := imageObj["sources"].([]interface{}); ok && len(sources) > 0 {
			if source, ok := sources[0].(map[string]interface{}); ok {
				if srcSet, ok := source["srcSet"].(string); ok {
					imageURL = srcSet
				}
			}
		}
	}

	// Extract part number for ID
	partNumber := ""
	if priceObj, ok := tile["price"].(map[string]interface{}); ok {
		partNumber, _ = priceObj["partNumber"].(string)
	}
	if partNumber == "" {
		if omniture, ok := tile["omnitureModel"].(map[string]interface{}); ok {
			partNumber, _ = omniture["partNumber"].(string)
		}
	}

	// Generate ID
	id := model.GenerateID(category, partNumber+title)

	// Parse specs from title
	cleanName := strings.TrimPrefix(title, "翻新 ")
	specs := ParseSpecs(title, "")

	// Parse detailed specs
	parsedSpecs := ParseProductSpecs(cleanName)
	specsDetailBytes, _ := json.Marshal(parsedSpecs.ToMap())

	// Use the category parameter directly, only normalize if it's a generic value
	// This preserves the correct category from the scrape URL
	normalizedCategory := category
	if category == "HomePod" || category == "AirPods" || category == "Apple TV" || category == "Accessories" {
		normalizedCategory = "Accessory"
	}

	product := &model.Product{
		ID:          id,
		Name:        cleanName,
		Category:    normalizedCategory,
		Region:      region,
		Price:       price,
		OriginalPrice: originalPrice,
		Discount:    discount,
		ImageURL:    imageURL,
		ProductURL:  productURL,
		Specs:       specs,
		SpecsDetail: string(specsDetailBytes),
		StockStatus: "available",
		// ValueScore will be calculated by SQLiteStore based on historical data
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
	}

	return product
}

// ScrapeProductDetails fetches additional details from a product's detail page
func (s *AppleScraper) ScrapeProductDetails(product *model.Product) *model.Product {
	if product.ProductURL == "" {
		return product
	}

	// Use FetchDetail for detail pages with better timeout and retry
	detailHTML, err := s.client.FetchDetail(product.ProductURL)
	if err != nil {
		// Fallback to regular Fetch with retry
		detailHTML, err = s.client.Fetch(product.ProductURL)
		if err != nil {
			return product
		}
	}

	// Extract description from the detail page
	description := s.extractDescription(detailHTML)

	// Extract detailed specs from the detail page
	detailedSpecs := s.parseSpecItems(detailHTML)

	// Parse existing specs_detail if any
	existingSpecs := make(map[string]interface{})
	if product.SpecsDetail != "" {
		_ = json.Unmarshal([]byte(product.SpecsDetail), &existingSpecs)
	}

	// Merge specs: detailed specs take precedence, then existing specs
	mergedSpecs := make(map[string]interface{})
	for k, v := range existingSpecs {
		mergedSpecs[k] = v
	}
	for k, v := range detailedSpecs {
		mergedSpecs[k] = v
	}

	// Always extract critical fields from description (memory, storage, camera, etc.)
	// These are often missing from the detailed specs but present in description
	if description != "" {
		descSpecs := s.extractSpecsFromDescription(description)
		// Only add fields that don't already exist in mergedSpecs
		for k, v := range descSpecs {
			if _, exists := mergedSpecs[k]; !exists {
				mergedSpecs[k] = v
			}
		}
	}

	// Marshal merged specs
	specsDetailBytes, _ := json.Marshal(mergedSpecs)

	// Update product with fetched details
	product.Description = description
	product.SpecsDetail = string(specsDetailBytes)

	return product
}

// extractDescription extracts the product description/overview from the detail page
func (s *AppleScraper) extractDescription(html string) string {
	// Apple uses multiple patterns for descriptions across different locales

	// Pattern 1: Look for meta description tag (most reliable for most pages)
	metaDescPattern := `<meta name="description" content="`
	metaStart := strings.Index(html, metaDescPattern)
	if metaStart != -1 {
		contentStart := metaStart + len(metaDescPattern)
		contentEnd := strings.Index(html[contentStart:], `"`)
		if contentEnd > 0 && contentEnd < 500 {
			desc := strings.TrimSpace(html[contentStart : contentStart+contentEnd])
			// Filter out invalid descriptions
			if desc != "" && len(desc) > 15 && len(desc) < 500 &&
			   !strings.HasPrefix(desc, "http") && !strings.Contains(desc, "ziyuan.baidu.com") &&
			   !strings.Contains(desc, "href=") && !strings.Contains(desc, "<") {
				return s.cleanHTML(desc)
			}
		}
	}

	// Pattern 2: Look for og:description meta tag (often has better descriptions)
	ogDescPattern := `<meta property="og:description" content="`
	ogStart := strings.Index(html, ogDescPattern)
	if ogStart != -1 {
		contentStart := ogStart + len(ogDescPattern)
		contentEnd := strings.Index(html[contentStart:], `"`)
		if contentEnd > 0 && contentEnd < 500 {
			desc := strings.TrimSpace(html[contentStart : contentStart+contentEnd])
			if desc != "" && len(desc) > 15 && len(desc) < 500 {
				return s.cleanHTML(desc)
			}
		}
	}

	// Pattern 3: Try Twitter description
	twitterDescPattern := `<meta name="twitter:description" content="`
	twitterStart := strings.Index(html, twitterDescPattern)
	if twitterStart != -1 {
		contentStart := twitterStart + len(twitterDescPattern)
		contentEnd := strings.Index(html[contentStart:], `"`)
		if contentEnd > 0 && contentEnd < 500 {
			desc := strings.TrimSpace(html[contentStart : contentStart+contentEnd])
			if desc != "" && len(desc) > 15 && len(desc) < 500 {
				return s.cleanHTML(desc)
			}
		}
	}

	// Pattern 4: Try to find product tagline or hero text
	// Look for specific Apple class patterns used for taglines
	taglinePatterns := []string{
		`"headline":`,
		`"tagline":`,
		`class="headline"`,
		`data-hero-headline`,
	}
	for _, pattern := range taglinePatterns {
		idx := strings.Index(html, pattern)
		if idx != -1 {
			contentStart := idx + len(pattern)
			// Find the next quote or bracket
			for i := contentStart; i < len(html) && i < contentStart+50; i++ {
				if html[i] == '"' || html[i] == '\'' || html[i] == '>' {
					contentStart = i + 1
					break
				}
			}
			// Find end
			for i := contentStart; i < len(html) && i < contentStart+300; i++ {
				if html[i] == '"' || html[i] == '\'' || html[i] == '<' || html[i] == '}' {
					desc := strings.TrimSpace(html[contentStart:i])
					if len(desc) > 10 && len(desc) < 200 {
						return s.cleanHTML(desc)
					}
					break
				}
			}
		}
	}

	// Pattern 5: Extract from JSON-LD structured data
	jsonLdStart := strings.Index(html, `"description":`)
	if jsonLdStart != -1 {
		contentStart := jsonLdStart + 15 // Skip past "description":
		contentEnd := strings.Index(html[contentStart:], `"`)
		if contentEnd > 0 && contentEnd < 300 {
			desc := strings.TrimSpace(html[contentStart : contentStart+contentEnd])
			if desc != "" && len(desc) > 20 && len(desc) < 300 {
				return s.cleanHTML(desc)
			}
		}
	}

	return ""
}

// cleanHTML removes HTML entities and cleans up text
func (s *AppleScraper) cleanHTML(text string) string {
	// Replace common HTML entities
	replacements := map[string]string{
		"&amp;":  "&",
		"&quot;": "\"",
		"&nbsp;": " ",
		"&lt;":   "<",
		"&gt;":   ">",
		"&#39;":  "'",
		"&#10;":  "\n", // Newline
		"&#13;":  "",  // Carriage return
		"\u00a0": " ", // Non-breaking space
		"\u200b": "",  // Zero-width space
	}
	cleaned := text
	for old, new := range replacements {
		cleaned = strings.ReplaceAll(cleaned, old, new)
	}
	return strings.TrimSpace(cleaned)
}

// cleanDescription removes HTML tags and excess whitespace from description
func (s *AppleScraper) cleanDescription(html string) string {
	// Remove HTML tags
	cleaned := strings.ReplaceAll(html, "<[^>]*>", " ")
	// Remove extra whitespace
	lines := strings.Fields(cleaned)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, " ")
}

// extractDetailedSpecs extracts detailed specifications from the detail page
func (s *AppleScraper) extractDetailedSpecs(html string) string {
	// Look for the specs section
	specsStart := strings.Index(html, "id=\"specs\"")
	if specsStart == -1 {
		// Try alternative pattern
		specsStart = strings.Index(html, "id=\"techspecs\"")
	}
	if specsStart == -1 {
		return ""
	}

	// Find the content after the specs div
	contentStart := specsStart + 12
	// Look for the closing tag
	depth := 0
	i := contentStart
	maxLen := len(html)
	if i >= maxLen {
		return ""
	}

	for i < maxLen {
		if strings.HasPrefix(html[i:], "<div") || strings.HasPrefix(html[i:], "<section") {
			depth++
			i += 4
		} else if strings.HasPrefix(html[i:], "</div>") || strings.HasPrefix(html[i:], "</section>") {
			depth--
			if depth == 0 {
				break
			}
			i += 6
		} else {
			i++
		}
	}

	// Extract the content
	content := html[contentStart:i]

	// Parse out individual spec items
	specs := s.parseSpecItems(content)
	if len(specs) == 0 {
		return ""
	}

	// Return as JSON string
	specsBytes, _ := json.Marshal(specs)
	return string(specsBytes)
}

// parseSpecItems parses individual specification items from HTML content
func (s *AppleScraper) parseSpecItems(html string) map[string]interface{} {
	specs := make(map[string]interface{})

	// Helper to check and add spec
	addSpec := func(key, value string) {
		if value != "" {
			specs[key] = value
		}
	}

	// Extract screen size / display (支持多种格式)
	screenPatterns := []struct {
		pattern string
		format  string
	}{
		{`(\d+(?:\.\d+)?)英寸.*?Liquid 视网膜.*?(\d+) x (\d+)`, "%s英寸 Liquid Retina XDR (%sx%s)"},
		{`(\d+(?:\.\d+)?)英寸.*?Liquid 视网膜.*?(\d+) x (\d+)`, "%s英寸 Liquid Retina (%sx%s)"},
		{`(\d+(?:\.\d+)?)英寸.*?Retina.*?(\d+) x (\d+)`, "%s英寸 Retina (%sx%s)"},
		{`(\d+(?:\.\d+)?)英寸`, "%s英寸"},
		{`(\d+(?:\.\d+)?)["\s]*Liquid`, "%s英寸 Liquid Retina"},
	}
	for _, sp := range screenPatterns {
		re := regexp.MustCompile(sp.pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			if len(match) > 3 {
				addSpec("display", fmt.Sprintf(sp.format, match[1], match[2], match[3]))
			} else {
				addSpec("display", fmt.Sprintf(sp.format, match[1]))
			}
			break
		}
	}

	// Extract memory (统一内存 / memory)
	memPatterns := []struct {
		pattern string
		format  string
	}{
		{`(\d+)[\s\xa0]*GB[\s\xa0]*统一[\s\xa0]*内存`, "%sGB 统一内存"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*内存`, "%sGB 内存"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*unified[\s\xa0]*memory`, "%sGB Unified Memory"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*memory`, "%sGB Memory"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*RAM`, "%sGB RAM"},
	}
	for _, mp := range memPatterns {
		re := regexp.MustCompile(mp.pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			addSpec("memory", fmt.Sprintf(mp.format, match[1]))
			break
		}
	}

	// Extract storage (SSD / solid state drive)
	storagePatterns := []struct {
		pattern string
		format  string
	}{
		{`(\d+)[\s\xa0]*TB[\s\xa0]*固态硬盘`, "%sTB 固态硬盘"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*固态硬盘`, "%sGB 固态硬盘"},
		{`(\d+)[\s\xa0]*TB[\s\xa0]*SSD`, "%sTB SSD"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*SSD`, "%sGB SSD"},
		{`(\d+)[\s\xa0]*TB[\s\xa0]*storage`, "%sTB Storage"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*storage`, "%sGB Storage"},
	}
	for _, sp := range storagePatterns {
		re := regexp.MustCompile(sp.pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			addSpec("storage", fmt.Sprintf(sp.format, match[1]))
			break
		}
	}

	// Extract chip (Apple Silicon)
	chipPatterns := []string{
		`M4\s*(?:Max|Pro)?`,
		`M3\s*(?:Max|Pro)?`,
		`M2\s*(?:Max|Pro|Ultra)?`,
		`M1\s*(?:Max|Pro|Ultra)?`,
	}
	for _, pattern := range chipPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(html); match != "" {
			addSpec("chip", strings.TrimSpace(match))
			break
		}
	}

	// Extract connectivity (Wi-Fi, Cellular)
	if strings.Contains(html, "Wi-Fi") && strings.Contains(html, "蜂窝") {
		addSpec("connectivity", "Wi-Fi + 蜂窝网络")
	} else if strings.Contains(html, "Wi-Fi") && strings.Contains(html, "Cellular") {
		addSpec("connectivity", "Wi-Fi + Cellular")
	} else if strings.Contains(html, "Wi-Fi") {
		addSpec("connectivity", "Wi-Fi")
	}

	// Extract camera info
	cameraPatterns := []struct {
		pattern string
		format  string
	}{
		{`(\d+)[\s\xa0]*p[\s\xa0]*FaceTime[\s\xa0]*高清[\s\xa0]*摄像头`, "%sp FaceTime 高清摄像头"},
		{`(\d+)[\s\xa0]*p[\s\xa0]*FaceTime[\s\xa0]*摄像头`, "%sp FaceTime 摄像头"},
		{`(\d+)[\s\xa0]*万像素`, "%s万像素"},
		{`1080p[\s\xa0]*FaceTime`, "1080p FaceTime 摄像头"},
		{`720p[\s\xa0]*FaceTime`, "720p FaceTime 摄像头"},
	}
	for _, cp := range cameraPatterns {
		re := regexp.MustCompile(cp.pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			addSpec("camera", fmt.Sprintf(cp.format, match[1]))
			break
		} else if match := re.FindString(html); match != "" {
			addSpec("camera", strings.TrimSpace(match))
			break
		}
	}

	// Extract Touch ID
	if strings.Contains(html, "触控 ID") || strings.Contains(html, "Touch ID") {
		addSpec("touch_id", "触控 ID")
	}

	// Extract Face ID
	if strings.Contains(html, "面容 ID") || strings.Contains(html, "Face ID") {
		addSpec("face_id", "面容 ID")
	}

	// Extract ports/connections
	if strings.Contains(html, "雷雳 4") || strings.Contains(html, "Thunderbolt 4") {
		if count := regexp.MustCompile(`雷霆[\s\xa0]*4|Thunderbolt[\s\xa0]*4`).FindAllStringIndex(html, -1); len(count) > 0 {
			portCount := len(count)
			if portCount == 3 {
				addSpec("ports", "三个雷雳 4 (USB-C) 端口")
			} else if portCount == 2 {
				addSpec("ports", "两个雷霆 4 (USB-C) 端口")
			} else {
				addSpec("ports", "雷雳 4 (USB-C)")
			}
		}
	} else if strings.Contains(html, "雷雳 3") || strings.Contains(html, "Thunderbolt 3") {
		addSpec("ports", "雷雳 3 (USB-C)")
	} else if strings.Contains(html, "USB-C") {
		addSpec("ports", "USB-C")
	}

	// Extract initial release date (最初发布于)
	releasePattern := regexp.MustCompile(`(?:最初发布于|Initial release|Released)\s*[：:]?\s*(\d{4})\s*年\s*(\d{1,2})\s*月`)
	if match := releasePattern.FindStringSubmatch(html); len(match) > 2 {
		addSpec("release_date", fmt.Sprintf("%s年%s月", match[1], match[2]))
	}

	// Extract battery info
	batteryPatterns := []struct {
		pattern string
		format  string
	}{
		{`最长\s*(\d+)\s*小时`, "最长 %s 小时"},
		{`up to\s*(\d+)\s*hours`, "最长 %s 小时"},
	}
	for _, bp := range batteryPatterns {
		re := regexp.MustCompile(bp.pattern)
		if match := re.FindStringSubmatch(html); len(match) > 1 {
			addSpec("battery", fmt.Sprintf(bp.format, match[1]))
			break
		}
	}

	// Extract keyboard type
	if strings.Contains(html, "妙控键盘") || strings.Contains(html, "Magic Keyboard") {
		addSpec("keyboard", "妙控键盘")
	}

	return specs
}

// extractSpecsFromDescription extracts specs from the description text
func (s *AppleScraper) extractSpecsFromDescription(desc string) map[string]interface{} {
	specs := make(map[string]interface{})

	// Extract memory from description
	memPatterns := []string{
		`(\d+)[\s\xa0]*GB[\s\xa0]*统一[\s\xa0]*内存`,
		`(\d+)[\s\xa0]*GB[\s\xa0]*内存`,
		`(\d+)[\s\xa0]*GB[\s\xa0]*unified[\s\xa0]*memory`,
		`(\d+)[\s\xa0]*GB[\s\xa0]*memory`,
		`(\d+)[\s\xa0]*GB[\s\xa0]*RAM`,
	}
	for _, pattern := range memPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(desc); len(match) > 1 {
			specs["memory"] = match[1] + "GB"
			break
		}
	}

	// Extract storage from description - FIXED: capture unit in regex
	storagePatterns := []struct {
		pattern string
		unit    string
	}{
		{`(\d+)[\s\xa0]*TB[\s\xa0]*固态[\s\xa0]*硬盘`, "TB"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*固态[\s\xa0]*硬盘`, "GB"},
		{`(\d+)[\s\xa0]*TB[\s\xa0]*SSD`, "TB"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*SSD`, "GB"},
		{`(\d+)[\s\xa0]*TB[\s\xa0]*storage`, "TB"},
		{`(\d+)[\s\xa0]*GB[\s\xa0]*storage`, "GB"},
	}
	for _, sp := range storagePatterns {
		re := regexp.MustCompile(sp.pattern)
		if match := re.FindStringSubmatch(desc); len(match) > 1 {
			specs["storage"] = match[1] + sp.unit
			break
		}
	}

	// Extract chip from description (as fallback, usually in specs_detail)
	chipPatterns := []string{
		`M[1-4]\s*(?:Max|Pro|Ultra)?`,
	}
	for _, pattern := range chipPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(desc); match != "" {
			specs["chip"] = strings.TrimSpace(match)
			break
		}
	}

	// Extract screen size from description
	screenRe := regexp.MustCompile(`(\d+(?:\.\d+)?)["\s]*英寸`)
	if match := screenRe.FindStringSubmatch(desc); len(match) > 1 {
		specs["screen_size"] = match[1] + "英寸"
	}

	// Extract camera from description
	cameraPatterns := []string{
		`(\d+)\s*MP\s*Center\s*Stage`,
		`(\d+)\s*MP`,
		`(\d+)\s*万像素`,
	}
	for _, pattern := range cameraPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(desc); len(match) > 1 {
			specs["camera"] = match[1] + "MP"
			break
		}
	}

	// Extract Touch ID
	if strings.Contains(desc, "触控 ID") || strings.Contains(desc, "Touch ID") {
		specs["touch_id"] = "触控 ID"
	}

	// Extract Face ID
	if strings.Contains(desc, "面容 ID") || strings.Contains(desc, "Face ID") {
		specs["face_id"] = "面容 ID"
	}

	// Extract ports
	if strings.Contains(desc, "雷霆 5") || strings.Contains(desc, "雷雳 5") {
		specs["ports"] = "雷雳 5"
	} else if strings.Contains(desc, "雷霆 4") || strings.Contains(desc, "雷雳 4") {
		specs["ports"] = "雷雳 4"
	} else if strings.Contains(desc, "Thunderbolt") {
		specs["ports"] = "Thunderbolt"
	}

	return specs
}
