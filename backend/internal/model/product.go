package model

import "time"

// Product represents an Apple refurbished product
type Product struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Category    string    `json:"category" db:"category"`       // Mac, iPad, iPhone, Watch, Accessory
	Region      string    `json:"region" db:"region"`           // cn, hk
	Price       float64   `json:"price" db:"price"`
	OriginalPrice float64 `json:"original_price" db:"original_price"`
	Discount    float64   `json:"discount" db:"discount"`
	ImageURL    string    `json:"image_url" db:"image_url"`
	ProductURL  string    `json:"product_url" db:"product_url"`
	Specs       string    `json:"specs" db:"specs"`
	SpecsDetail string    `json:"specs_detail,omitempty" db:"specs_detail"` // JSON string of parsed specs
	Description string    `json:"description,omitempty" db:"description"` // Product overview/description
	StockStatus string    `json:"stock_status" db:"stock_status"` // available, sold_out, limited

	// Value-based scoring (replaces AI-based scoring)
	ValueScore  float64  `json:"value_score" db:"value_score"` // 0-100, based on historical data
	LowestPrice float64  `json:"lowest_price,omitempty" db:"lowest_price"`
	HighestPrice float64 `json:"highest_price,omitempty" db:"highest_price"`
	PriceTrend  string   `json:"price_trend,omitempty" db:"price_trend"` // falling, rising, stable

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// PriceHistory represents a price change record
type PriceHistory struct {
	ProductID string    `json:"product_id"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Discount  float64   `json:"discount"`
}

// Subscription represents a user subscription for price notifications
type Subscription struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	BarkKey    string    `json:"bark_key,omitempty"`
	Email      string    `json:"email,omitempty"`
	TargetPrice float64  `json:"target_price,omitempty"` // Target price for alert (0 = any drop)
	CreatedAt  time.Time `json:"created_at"`
}

// NewArrivalSubscription represents a subscription for new product arrival notifications
type NewArrivalSubscription struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`        // User-defined name for this subscription
	Categories []string  `json:"categories"`  // Filter by categories (empty = all)
	MaxPrice   float64   `json:"max_price"`   // Maximum price filter (0 = no limit)
	MinPrice   float64   `json:"min_price"`   // Minimum price filter (0 = no limit)
	Keywords   []string  `json:"keywords"`    // Product name must contain these keywords
	BarkKey    string    `json:"bark_key,omitempty"`
	Email      string    `json:"email,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

// Stats represents system statistics
type Stats struct {
	TotalProducts      int            `json:"total_products"`
	AvailableProducts int            `json:"available_products"`
	Categories        map[string]int `json:"categories"`
	LastScrapeTime    time.Time      `json:"last_scrape_time"`
	TotalSubscriptions int           `json:"total_subscriptions"`
}

// GenerateID creates a unique product ID based on category and specs
func GenerateID(category, specs string) string {
	// Simple hash-based ID generation
	// In production, you might use a more sophisticated approach
	return "cn:" + category + ":" + hashString(specs)
}

func hashString(s string) string {
	// Improved hash function using FNV-1a algorithm for better distribution
	const prime32 = 16777619
	hash32 := uint32(2166136261)

	for i, c := range s {
		if i > 100 { // Hash first 100 chars
			break
		}
		hash32 ^= uint32(c)
		hash32 *= prime32
	}

	// Convert to base36 string for compact representation
	const charset = "0123456789abcdefghijklmnopqrstuvwxyz"
	var result [8]byte
	for i := 0; i < 8; i++ {
		result[i] = charset[hash32%36]
		hash32 /= 36
	}
	return string(result[:])
}
