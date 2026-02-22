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
	BarkKey    string    `json:"bark_key"`
	TargetPrice float64  `json:"target_price,omitempty"` // Target price for alert (0 = any drop)
	CreatedAt  time.Time `json:"created_at"`
}

// NewArrivalSubscription represents a subscription for new product arrival notifications
type NewArrivalSubscription struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`                // User-defined name for this subscription
	Description       string    `json:"description,omitempty"` // User notes
	Categories        []string  `json:"categories"`          // Filter by categories (empty = all)
	Models            []string  `json:"models,omitempty"`            // Filter by product models (MacBook Pro, iPad Pro, etc.)
	Chips             []string  `json:"chips,omitempty"`             // Filter by chip models (M1 Pro, M2 Max, etc.)
	Storages          []string  `json:"storages,omitempty"`          // Filter by storage (256GB, 512GB, etc.)
	Memories          []string  `json:"memories,omitempty"`          // Filter by memory (8GB, 16GB, etc.)
	StockStatuses     []string  `json:"stock_statuses,omitempty"`     // Filter by stock status (available, limited)
	MaxPrice          float64   `json:"max_price"`           // Maximum price filter (0 = no limit)
	MinPrice          float64   `json:"min_price"`           // Minimum price filter (0 = no limit)
	Keywords          []string  `json:"keywords"`            // Product name must contain these keywords
	BarkKey           string    `json:"bark_key"`
	NotifiedProductIDs string    `json:"notified_product_ids"` // JSON array of product IDs that have been notified
	Enabled           bool      `json:"enabled"`
	Paused            bool      `json:"paused"`                        // Paused by user
	NotificationCount int       `json:"notification_count"`             // Number of notifications sent
	LastNotifiedAt    time.Time `json:"last_notified_at,omitempty"`    // Last notification time
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

// NotificationHistory represents a record of sent notification
type NotificationHistory struct {
	ID               string    `json:"id"`
	SubscriptionID   string    `json:"subscription_id"`
	ProductID        string    `json:"product_id"`
	ProductName      string    `json:"product_name"`
	ProductCategory  string    `json:"product_category"`
	ProductPrice     float64   `json:"product_price"`
	ProductImageURL  string    `json:"product_image_url"`
	ProductSpecs     string    `json:"product_specs"`     // JSON: parsed specs
	NotificationType string    `json:"notification_type"` // new_arrival, price_drop
	Status           string    `json:"status"`            // sent, failed
	ErrorMessage     string    `json:"error_message,omitempty"`
	BarkKey          string    `json:"-"`                 // Full key for filtering, not exposed in JSON
	BarkKeyMasked    string    `json:"bark_key_masked"`
	CreatedAt        time.Time `json:"created_at"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
}

// ParsedSpecs represents parsed product specifications
type ParsedSpecs struct {
	Chip         string `json:"chip,omitempty"`         // M1 Pro, M2 Max, etc.
	Memory       string `json:"memory,omitempty"`       // 8GB, 16GB, etc.
	Storage      string `json:"storage,omitempty"`       // 256GB, 512GB, etc.
	ScreenSize   string `json:"screen_size,omitempty"`  // 14", 16", etc.
	Color        string `json:"color,omitempty"`         // 深空黑, 银色, etc.
}

// ScraperStatus represents the scraper health status
type ScraperStatus struct {
	LastScrapeTime   time.Time `json:"last_scrape_time"`
	LastScrapeStatus string    `json:"last_scrape_status"` // success, failed, running, never
	LastScrapeError  string    `json:"last_scrape_error,omitempty"`
	ProductsScraped  int       `json:"products_scraped"`
	Duration         int64     `json:"duration_ms"`
}

// Stats represents system statistics
type Stats struct {
	TotalProducts      int            `json:"total_products"`
	AvailableProducts  int            `json:"available_products"`
	Categories         map[string]int `json:"categories"`
	LastScrapeTime     time.Time      `json:"last_scrape_time"`
	TotalSubscriptions int            `json:"total_subscriptions"`
	ScraperStatus      *ScraperStatus `json:"scraper_status,omitempty"`
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
