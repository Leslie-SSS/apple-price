package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	barkAPIURL = "https://api.day.app"
)

// BarkService handles Bark notifications
type BarkService struct {
	client    *http.Client
	isEnabled bool
}

// NewBarkService creates a new Bark notification service
func NewBarkService() *BarkService {
	return &BarkService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		isEnabled: true,
	}
}

// Disable disables the Bark service
func (b *BarkService) Disable() {
	b.isEnabled = false
}

// SendNotification sends a Bark notification
func (b *BarkService) SendNotification(key, title, content string) error {
	if !b.isEnabled {
		return nil
	}

	if key == "" {
		return fmt.Errorf("bark key is empty")
	}

	// URL encode the title and content
	title = url.QueryEscape(title)
	content = url.QueryEscape(content)

	// Build URL: https://api.day.app/{key}/{title}/{content}
	barkURL := fmt.Sprintf("%s/%s/%s/%s", barkAPIURL, key, title, content)

	req, err := http.NewRequest("GET", barkURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// SendPriceChangeNotification sends a price change notification
func (b *BarkService) SendPriceChangeNotification(key, productName string, oldPrice, newPrice float64, productURL string) error {
	title := "🍎 苹果翻新价格变动"
	content := fmt.Sprintf("%s 价格从 %.2f 变为 %.2f，点击查看详情",
		productName, oldPrice, newPrice)

	// Add URL to content if provided
	if productURL != "" {
		content += fmt.Sprintf("?url=%s", url.QueryEscape(productURL))
	}

	return b.SendNotification(key, title, content)
}

// SendStockNotification sends a stock availability notification
func (b *BarkService) SendStockNotification(key, productName string, stockStatus string, productURL string) error {
	title := "🍎 苹果翻新库存提醒"
	content := fmt.Sprintf("%s 状态更新为: %s", productName, stockStatus)

	if productURL != "" {
		content += fmt.Sprintf("?url=%s", url.QueryEscape(productURL))
	}

	return b.SendNotification(key, title, content)
}

// SendNewArrivalNotification sends a new product arrival notification
func (b *BarkService) SendNewArrivalNotification(key, productName string, price float64, category, productURL string) error {
	title := "🆕 苹果翻新新品上架"
	content := fmt.Sprintf("[%s] %s 到货了！价格: ¥%.0f", category, productName, price)

	if productURL != "" {
		content += fmt.Sprintf("?url=%s", url.QueryEscape(productURL))
	}

	return b.SendNotification(key, title, content)
}

// SendBatchNotification sends a batch notification for multiple products
func (b *BarkService) SendBatchNotification(key string, changes []PriceChange) error {
	if len(changes) == 0 {
		return nil
	}

	title := "🍎 苹果翻新价格汇总"
	var content strings.Builder

	content.WriteString(fmt.Sprintf("发现 %d 个价格变动\n\n", len(changes)))

	for i, change := range changes {
		if i >= 5 { // Limit to 5 items
			content.WriteString(fmt.Sprintf("...还有 %d 个产品", len(changes)-5))
			break
		}
		content.WriteString(fmt.Sprintf("%s: %.2f → %.2f\n",
			change.ProductName, change.OldPrice, change.NewPrice))
	}

	return b.SendNotification(key, title, content.String())
}

// ValidateKey validates a Bark key
func (b *BarkService) ValidateKey(key string) bool {
	if key == "" {
		return false
	}

	// Basic validation - Bark keys are typically alphanumeric
	// and vary in length, but should not contain spaces
	if strings.Contains(key, " ") {
		return false
	}

	return true
}

// PriceChange represents a price change for batch notifications
type PriceChange struct {
	ProductName string
	OldPrice    float64
	NewPrice    float64
}
