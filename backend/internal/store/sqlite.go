package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"apple-price/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore manages product data using SQLite database
type SQLiteStore struct {
	db            *sql.DB
	mu            sync.RWMutex
	dataDir       string
	lastScrapeTime time.Time
}

// NewSQLite creates a new SQLiteStore instance
func NewSQLite(dataDir string) (*SQLiteStore, error) {
	dbPath := filepath.Join(dataDir, "apple-price.db")

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with WAL mode and foreign keys enabled
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &SQLiteStore{
		db:      db,
		dataDir: dataDir,
	}

	// Run migrations
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

// migrate creates tables and indexes
func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		region TEXT NOT NULL,
		price REAL NOT NULL,
		original_price REAL NOT NULL,
		discount REAL NOT NULL,
		image_url TEXT,
		product_url TEXT NOT NULL,
		specs TEXT,
		specs_detail TEXT,
		description TEXT,
		stock_status TEXT NOT NULL DEFAULT 'available',
		value_score REAL DEFAULT 0,
		lowest_price REAL,
		highest_price REAL,
		price_trend TEXT DEFAULT 'stable',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS price_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id TEXT NOT NULL,
		price REAL NOT NULL,
		discount REAL NOT NULL,
		recorded_at INTEGER NOT NULL,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS subscriptions (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL,
		bark_key TEXT,
		email TEXT,
		target_price REAL DEFAULT 0,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS new_arrival_subscriptions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		categories TEXT,
		models TEXT,
		chips TEXT,
		storages TEXT,
		memories TEXT,
		stock_statuses TEXT,
		max_price REAL DEFAULT 0,
		min_price REAL DEFAULT 0,
		keywords TEXT,
		bark_key TEXT,
		notified_product_ids TEXT DEFAULT '[]',
		enabled INTEGER DEFAULT 1,
		paused INTEGER DEFAULT 0,
		notification_count INTEGER DEFAULT 0,
		last_notified_at INTEGER,
		created_at INTEGER NOT NULL,
		updated_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS notification_history (
		id TEXT PRIMARY KEY,
		subscription_id TEXT NOT NULL,
		product_id TEXT NOT NULL,
		product_name TEXT NOT NULL,
		product_category TEXT NOT NULL,
		product_price REAL NOT NULL,
		product_image_url TEXT,
		product_specs TEXT,
		notification_type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'sent',
		error_message TEXT,
		bark_key TEXT NOT NULL DEFAULT '',
		bark_key_masked TEXT,
		created_at INTEGER NOT NULL,
		read_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS scraper_status (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		last_scrape_time INTEGER,
		last_scrape_status TEXT DEFAULT 'never',
		last_scrape_error TEXT,
		products_scraped INTEGER DEFAULT 0,
		duration_ms INTEGER DEFAULT 0,
		updated_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
	CREATE INDEX IF NOT EXISTS idx_products_region ON products(region);
	CREATE INDEX IF NOT EXISTS idx_products_stock_status ON products(stock_status);
	CREATE INDEX IF NOT EXISTS idx_products_value_score ON products(value_score DESC);
	CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_price_history_product_id ON price_history(product_id);
	CREATE INDEX IF NOT EXISTS idx_price_history_product_recorded ON price_history(product_id, recorded_at DESC);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_product_id ON subscriptions(product_id);
	CREATE INDEX IF NOT EXISTS idx_new_arrival_subscriptions_enabled ON new_arrival_subscriptions(enabled);
	CREATE INDEX IF NOT EXISTS idx_notification_history_subscription ON notification_history(subscription_id, created_at DESC);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	// Add specs_detail column if it doesn't exist (for existing databases)
	s.db.Exec(`ALTER TABLE products ADD COLUMN specs_detail TEXT`)

	// Add description column if it doesn't exist (for existing databases)
	s.db.Exec(`ALTER TABLE products ADD COLUMN description TEXT`)

	// Add target_price column to subscriptions if it doesn't exist (for existing databases)
	s.db.Exec(`ALTER TABLE subscriptions ADD COLUMN target_price REAL DEFAULT 0`)

	// Remove email column from subscriptions if it exists (migration)
	s.db.Exec(`ALTER TABLE subscriptions DROP COLUMN email`)

	// Add notified_product_ids column to new_arrival_subscriptions
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN notified_product_ids TEXT DEFAULT '[]'`)

	// Add new columns to new_arrival_subscriptions for enhanced filtering
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN description TEXT`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN chips TEXT`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN storages TEXT`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN memories TEXT`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN stock_statuses TEXT`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN models TEXT`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN paused INTEGER DEFAULT 0`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN notification_count INTEGER DEFAULT 0`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN last_notified_at INTEGER`)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions ADD COLUMN updated_at INTEGER`)

	// Remove email column from new_arrival_subscriptions if it exists (migration)
	s.db.Exec(`ALTER TABLE new_arrival_subscriptions DROP COLUMN email`)

	// SQLite doesn't support "IF NOT EXISTS" for ALTER TABLE, so we ignore the error
	// if the column already exists

	return nil
}

// GetAllProducts returns all products
func (s *SQLiteStore) GetAllProducts() []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, category, region, price, original_price, discount,
		       image_url, product_url, specs, specs_detail, description, stock_status, value_score,
		       lowest_price, highest_price, price_trend, created_at, updated_at
		FROM products
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return []*model.Product{}
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p := &model.Product{}
		var created, updated int64
		var lowest, highest sql.NullFloat64
		var trend sql.NullString
		var specsDetail, description sql.NullString

		err := rows.Scan(
			&p.ID, &p.Name, &p.Category, &p.Region, &p.Price, &p.OriginalPrice,
			&p.Discount, &p.ImageURL, &p.ProductURL, &p.Specs, &specsDetail, &description, &p.StockStatus,
			&p.ValueScore, &lowest, &highest, &trend, &created, &updated,
		)
		if err != nil {
			continue
		}

		if specsDetail.Valid {
			p.SpecsDetail = specsDetail.String
		}
		if description.Valid {
			p.Description = description.String
		}

		if lowest.Valid {
			p.LowestPrice = lowest.Float64
		}
		if highest.Valid {
			p.HighestPrice = highest.Float64
		}
		if trend.Valid {
			p.PriceTrend = trend.String
		}

		p.CreatedAt = time.Unix(created, 0)
		p.UpdatedAt = time.Unix(updated, 0)
		products = append(products, p)
	}

	return products
}

// GetProduct returns a product by ID
func (s *SQLiteStore) GetProduct(id string) (*model.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p := &model.Product{}
	var created, updated int64
	var lowest, highest sql.NullFloat64
	var trend sql.NullString
	var specsDetail, description sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, category, region, price, original_price, discount,
		       image_url, product_url, specs, specs_detail, description, stock_status, value_score,
		       lowest_price, highest_price, price_trend, created_at, updated_at
		FROM products WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &p.Category, &p.Region, &p.Price, &p.OriginalPrice,
		&p.Discount, &p.ImageURL, &p.ProductURL, &p.Specs, &specsDetail, &description, &p.StockStatus,
		&p.ValueScore, &lowest, &highest, &trend, &created, &updated,
	)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	if specsDetail.Valid {
		p.SpecsDetail = specsDetail.String
	}
	if description.Valid {
		p.Description = description.String
	}
	if lowest.Valid {
		p.LowestPrice = lowest.Float64
	}
	if highest.Valid {
		p.HighestPrice = highest.Float64
	}
	if trend.Valid {
		p.PriceTrend = trend.String
	}

	p.CreatedAt = time.Unix(created, 0)
	p.UpdatedAt = time.Unix(updated, 0)

	return p, true
}

// GetProductsByCategory returns products filtered by category
func (s *SQLiteStore) GetProductsByCategory(category string) []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, category, region, price, original_price, discount,
		       image_url, product_url, specs, specs_detail, description, stock_status, value_score,
		       lowest_price, highest_price, price_trend, created_at, updated_at
		FROM products WHERE category = ?
		ORDER BY updated_at DESC
	`, category)
	if err != nil {
		return []*model.Product{}
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p := &model.Product{}
		var created, updated int64
		var lowest, highest sql.NullFloat64
		var trend sql.NullString
		var specsDetail, description sql.NullString

		err := rows.Scan(
			&p.ID, &p.Name, &p.Category, &p.Region, &p.Price, &p.OriginalPrice,
			&p.Discount, &p.ImageURL, &p.ProductURL, &p.Specs, &specsDetail, &description, &p.StockStatus,
			&p.ValueScore, &lowest, &highest, &trend, &created, &updated,
		)
		if err != nil {
			continue
		}

		if specsDetail.Valid {
			p.SpecsDetail = specsDetail.String
		}
		if description.Valid {
			p.Description = description.String
		}
		if lowest.Valid {
			p.LowestPrice = lowest.Float64
		}
		if highest.Valid {
			p.HighestPrice = highest.Float64
		}
		if trend.Valid {
			p.PriceTrend = trend.String
		}

		p.CreatedAt = time.Unix(created, 0)
		p.UpdatedAt = time.Unix(updated, 0)
		products = append(products, p)
	}

	return products
}

// GetProductsByRegion returns products filtered by region
func (s *SQLiteStore) GetProductsByRegion(region string) []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, category, region, price, original_price, discount,
		       image_url, product_url, specs, specs_detail, description, stock_status, value_score,
		       lowest_price, highest_price, price_trend, created_at, updated_at
		FROM products WHERE region = ?
		ORDER BY updated_at DESC
	`, region)
	if err != nil {
		return []*model.Product{}
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p := &model.Product{}
		var created, updated int64
		var lowest, highest sql.NullFloat64
		var trend sql.NullString
		var specsDetail, description sql.NullString

		err := rows.Scan(
			&p.ID, &p.Name, &p.Category, &p.Region, &p.Price, &p.OriginalPrice,
			&p.Discount, &p.ImageURL, &p.ProductURL, &p.Specs, &specsDetail, &description, &p.StockStatus,
			&p.ValueScore, &lowest, &highest, &trend, &created, &updated,
		)
		if err != nil {
			continue
		}

		if specsDetail.Valid {
			p.SpecsDetail = specsDetail.String
		}
		if description.Valid {
			p.Description = description.String
		}
		if lowest.Valid {
			p.LowestPrice = lowest.Float64
		}
		if highest.Valid {
			p.HighestPrice = highest.Float64
		}
		if trend.Valid {
			p.PriceTrend = trend.String
		}

		p.CreatedAt = time.Unix(created, 0)
		p.UpdatedAt = time.Unix(updated, 0)
		products = append(products, p)
	}

	return products
}

// UpsertProduct adds or updates a product, returns true if price changed
func (s *SQLiteStore) UpsertProduct(product *model.Product) (priceChanged bool, oldPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if product exists
	var existingPrice sql.NullFloat64
	err := s.db.QueryRow("SELECT price FROM products WHERE id = ?", product.ID).Scan(&existingPrice)

	if err == sql.ErrNoRows {
		// New product
		product.CreatedAt = now
		priceChanged = false
		oldPrice = 0
	} else if err != nil {
		// Error
		return false, 0
	} else {
		// Existing product - always set oldPrice to distinguish from new products
		oldPrice = existingPrice.Float64

		if existingPrice.Float64 != product.Price {
			priceChanged = true

			// Add to history
			_, _ = s.db.Exec(`
				INSERT INTO price_history (product_id, price, discount, recorded_at)
				VALUES (?, ?, ?, ?)
			`, product.ID, existingPrice.Float64, product.Discount, now.Unix())
		}

			// Preserve created_at
		var created int64
		_ = s.db.QueryRow("SELECT created_at FROM products WHERE id = ?", product.ID).Scan(&created)
		product.CreatedAt = time.Unix(created, 0)

		// Preserve existing description and specs_detail if new ones are empty
		// This prevents the main scraper from overwriting data collected by detail scraper
		var existingDesc sql.NullString
		var existingSpecsDetail sql.NullString
		_ = s.db.QueryRow("SELECT description, specs_detail FROM products WHERE id = ?", product.ID).Scan(&existingDesc, &existingSpecsDetail)
		if product.Description == "" && existingDesc.Valid && existingDesc.String != "" {
			product.Description = existingDesc.String
		}
		if product.SpecsDetail == "" && existingSpecsDetail.Valid && existingSpecsDetail.String != "" {
			product.SpecsDetail = existingSpecsDetail.String
		}

		// Calculate value score based on history
		history := s.getPriceHistoryLocked(product.ID)
		product.ValueScore = s.CalculateValueScore(product, history)
		s.updateProductStats(product.ID, history)
	}

	product.UpdatedAt = now

	_, err = s.db.Exec(`
		INSERT INTO products (
			id, name, category, region, price, original_price, discount,
			image_url, product_url, specs, specs_detail, description, stock_status, value_score,
			lowest_price, highest_price, price_trend, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			category = excluded.category,
			region = excluded.region,
			price = excluded.price,
			original_price = excluded.original_price,
			discount = excluded.discount,
			image_url = excluded.image_url,
			product_url = excluded.product_url,
			specs = excluded.specs,
			specs_detail = excluded.specs_detail,
			description = excluded.description,
			stock_status = excluded.stock_status,
			value_score = excluded.value_score,
			lowest_price = excluded.lowest_price,
			highest_price = excluded.highest_price,
			price_trend = excluded.price_trend,
			updated_at = excluded.updated_at
	`, product.ID, product.Name, product.Category, product.Region, product.Price,
		product.OriginalPrice, product.Discount, product.ImageURL, product.ProductURL,
		product.Specs, product.SpecsDetail, product.Description, product.StockStatus, product.ValueScore,
		product.LowestPrice, product.HighestPrice, product.PriceTrend,
		product.CreatedAt.Unix(), product.UpdatedAt.Unix())

	if err != nil {
		fmt.Printf("[SQLiteStore] ERROR upserting product %s: %v\n", product.ID, err)
	} else if product.Description != "" {
		fmt.Printf("[SQLiteStore] Successfully upserted product %s with description: %d chars\n", product.ID, len(product.Description))
	}

	return priceChanged, oldPrice
}

// GetPriceHistory returns price history for a product
func (s *SQLiteStore) GetPriceHistory(productID string) []model.PriceHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getPriceHistoryLocked(productID)
}

// getPriceHistoryLocked returns price history WITHOUT acquiring lock (must be called with lock already held)
func (s *SQLiteStore) getPriceHistoryLocked(productID string) []model.PriceHistory {
	rows, err := s.db.Query(`
		SELECT product_id, price, discount, recorded_at
		FROM price_history
		WHERE product_id = ?
		ORDER BY recorded_at ASC
	`, productID)
	if err != nil {
		return []model.PriceHistory{}
	}
	defer rows.Close()

	var history []model.PriceHistory
	for rows.Next() {
		var h model.PriceHistory
		var recorded int64
		err := rows.Scan(&h.ProductID, &h.Price, &h.Discount, &recorded)
		if err != nil {
			continue
		}
		h.Timestamp = time.Unix(recorded, 0)
		h.ProductID = productID
		history = append(history, h)
	}

	return history
}

// GetCategories returns all unique categories
func (s *SQLiteStore) GetCategories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT DISTINCT category FROM products ORDER BY category")
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if rows.Scan(&cat) == nil {
			categories = append(categories, cat)
		}
	}

	return categories
}

// AddSubscription adds a new subscription
func (s *SQLiteStore) AddSubscription(sub *model.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (id, product_id, bark_key, target_price, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, sub.ID, sub.ProductID, sub.BarkKey, sub.TargetPrice, sub.CreatedAt.Unix())

	return err
}

// RemoveSubscription removes a subscription
func (s *SQLiteStore) RemoveSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM subscriptions WHERE id = ?", id)
	return err
}

// DeleteProductsByRegion deletes all products from a specific region
func (s *SQLiteStore) DeleteProductsByRegion(region string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec("DELETE FROM products WHERE region = ?", region)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// GetAllSubscriptions returns all subscriptions
func (s *SQLiteStore) GetAllSubscriptions() []*model.Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, product_id, bark_key, target_price, created_at
		FROM subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return []*model.Subscription{}
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		sub := &model.Subscription{}
		var created int64
		var targetPrice sql.NullFloat64
		err := rows.Scan(&sub.ID, &sub.ProductID, &sub.BarkKey, &targetPrice, &created)
		if err != nil {
			continue
		}
		if targetPrice.Valid {
			sub.TargetPrice = targetPrice.Float64
		}
		sub.CreatedAt = time.Unix(created, 0)
		subs = append(subs, sub)
	}

	return subs
}

// GetSubscriptionsByProduct returns all subscriptions for a product
func (s *SQLiteStore) GetSubscriptionsByProduct(productID string) []*model.Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, product_id, bark_key, target_price, created_at
		FROM subscriptions
		WHERE product_id = ?
		ORDER BY created_at DESC
	`, productID)
	if err != nil {
		return []*model.Subscription{}
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		sub := &model.Subscription{}
		var created int64
		var targetPrice sql.NullFloat64
		err := rows.Scan(&sub.ID, &sub.ProductID, &sub.BarkKey, &targetPrice, &created)
		if err != nil {
			continue
		}
		if targetPrice.Valid {
			sub.TargetPrice = targetPrice.Float64
		}
		sub.CreatedAt = time.Unix(created, 0)
		subs = append(subs, sub)
	}

	return subs
}

// UpdateLastScrapeTime updates the last scrape timestamp
func (s *SQLiteStore) UpdateLastScrapeTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastScrapeTime = t
}

// GetLastScrapeTime returns the last scrape timestamp
func (s *SQLiteStore) GetLastScrapeTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScrapeTime
}

// GetStats returns system statistics
func (s *SQLiteStore) GetStats() *model.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &model.Stats{
		Categories:         make(map[string]int),
		LastScrapeTime:     s.lastScrapeTime,
	}

	// Total products
	_ = s.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&stats.TotalProducts)

	// Available products
	_ = s.db.QueryRow("SELECT COUNT(*) FROM products WHERE stock_status = 'available'").Scan(&stats.AvailableProducts)

	// Categories
	rows, _ := s.db.Query("SELECT category, COUNT(*) FROM products GROUP BY category")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var count int
			if rows.Scan(&cat, &count) == nil {
				stats.Categories[cat] = count
			}
		}
	}

	// Total subscriptions
	_ = s.db.QueryRow("SELECT COUNT(*) FROM subscriptions").Scan(&stats.TotalSubscriptions)

	// Scraper status
	scraperStatus := &model.ScraperStatus{}
	var lastTime sql.NullInt64
	var scrapeErr sql.NullString
	var productsScraped sql.NullInt64
	var duration sql.NullInt64

	err := s.db.QueryRow(`
		SELECT last_scrape_time, last_scrape_status, last_scrape_error,
			   products_scraped, duration_ms
		FROM scraper_status WHERE id = 1
	`).Scan(&lastTime, &scraperStatus.LastScrapeStatus, &scrapeErr,
		&productsScraped, &duration)

	if err == nil {
		if lastTime.Valid {
			scraperStatus.LastScrapeTime = time.Unix(lastTime.Int64, 0)
		}
		if scrapeErr.Valid {
			scraperStatus.LastScrapeError = scrapeErr.String
		}
		if productsScraped.Valid {
			scraperStatus.ProductsScraped = int(productsScraped.Int64)
		}
		if duration.Valid {
			scraperStatus.Duration = duration.Int64
		}
		stats.ScraperStatus = scraperStatus
	} else {
		// No status record yet
		stats.ScraperStatus = &model.ScraperStatus{
			LastScrapeStatus: "never",
		}
	}

	return stats
}

// CalculateValueScore calculates value score based on historical data
// Note: Discount is fixed at 15% for Apple refurbished products, so we removed discount from scoring
func (s *SQLiteStore) CalculateValueScore(product *model.Product, history []model.PriceHistory) float64 {
	score := 50.0 // Base score

	// 1. Price trend score (0-35 points) - increased weight
	trendScore := s.trendScore(history)
	score += trendScore * 1.4

	// 2. Stock status score (0-20 points) - increased weight
	stockScore := s.stockScore(product.StockStatus)
	score += stockScore * 1.33

	// 3. Price position score (0-30 points) - increased weight
	positionScore := s.pricePositionScore(product.Price, history)
	score += positionScore * 1.5

	// 4. Age score (0-15 points) - increased weight
	ageScore := s.ageScore(product.CreatedAt)
	score += ageScore * 1.5

	// Cap at 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

func (s *SQLiteStore) trendScore(history []model.PriceHistory) float64 {
	if len(history) < 3 {
		return 0
	}

	recent := history[len(history)-3:]
	startPrice := recent[0].Price
	endPrice := recent[2].Price
	change := (endPrice - startPrice) / startPrice

	if change < -0.02 { // Fell more than 2%
		return 25
	} else if change < -0.01 { // Fell more than 1%
		return 20
	} else if change < 0 { // Falling
		return 15
	} else if change > 0.02 { // Rose more than 2%
		return 0
	}
	return 10 // Stable
}

func (s *SQLiteStore) stockScore(status string) float64 {
	switch status {
	case "available":
		return 15
	case "limited":
		return 10
	default:
		return 0
	}
}

func (s *SQLiteStore) pricePositionScore(currentPrice float64, history []model.PriceHistory) float64 {
	if len(history) == 0 {
		return 10
	}

	min, max := history[0].Price, history[0].Price
	for _, h := range history {
		if h.Price < min {
			min = h.Price
		}
		if h.Price > max {
			max = h.Price
		}
	}

	if max == min {
		return 10
	}

	position := (currentPrice - min) / (max - min)
	if position <= 0.1 {
		return 20
	} else if position <= 0.3 {
		return 15
	} else if position <= 0.5 {
		return 10
	} else if position <= 0.7 {
		return 5
	}
	return 0
}

func (s *SQLiteStore) ageScore(createdAt time.Time) float64 {
	days := time.Since(createdAt).Hours() / 24
	switch {
	case days <= 7:
		return 10
	case days <= 30:
		return 7
	case days <= 90:
		return 3
	default:
		return 0
	}
}

// updateProductStats updates lowest_price, highest_price, and price_trend
func (s *SQLiteStore) updateProductStats(productID string, history []model.PriceHistory) {
	if len(history) == 0 {
		return
	}

	min, max := history[0].Price, history[0].Price
	for _, h := range history {
		if h.Price < min {
			min = h.Price
		}
		if h.Price > max {
			max = h.Price
		}
	}

	// Determine trend
	trend := "stable"
	if len(history) >= 3 {
		recent := history[len(history)-3:]
		change := (recent[2].Price - recent[0].Price) / recent[0].Price
		if change < -0.02 {
			trend = "falling"
		} else if change > 0.02 {
			trend = "rising"
		}
	}

	s.db.Exec(`
		UPDATE products
		SET lowest_price = ?, highest_price = ?, price_trend = ?
		WHERE id = ?
	`, min, max, trend, productID)
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Save is a no-op for SQLite (data is persisted automatically)
// This method exists for compatibility with the old JSON store interface
func (s *SQLiteStore) Save() error {
	// SQLite persists data automatically, so this is a no-op
	// However, we can run a WAL checkpoint to optimize
	_, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// AddNewArrivalSubscription adds a new arrival subscription
func (s *SQLiteStore) AddNewArrivalSubscription(sub *model.NewArrivalSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Debug: Check what we're receiving
	fmt.Fprintf(os.Stderr, "[DEBUG] Categories=%v, len=%d\n", sub.Categories, len(sub.Categories))
	for i, cat := range sub.Categories {
		fmt.Fprintf(os.Stderr, "[DEBUG]   Categories[%d] = %q (len=%d)\n", i, cat, len(cat))
	}

	// Use json.Marshal for proper JSON encoding
	categoriesJSON, _ := json.Marshal(sub.Categories)
	modelsJSON, _ := json.Marshal(sub.Models)
	chipsJSON, _ := json.Marshal(sub.Chips)
	storagesJSON, _ := json.Marshal(sub.Storages)
	memoriesJSON, _ := json.Marshal(sub.Memories)
	stockStatusesJSON, _ := json.Marshal(sub.StockStatuses)
	keywordsJSON, _ := json.Marshal(sub.Keywords)

	// Debug: Check what JSON marshal produces
	fmt.Fprintf(os.Stderr, "[DEBUG] json.Marshal result: %q (len=%d)\n", string(categoriesJSON), len(categoriesJSON))

	enabled := 1
	if !sub.Enabled {
		enabled = 0
	}

	paused := 0
	if sub.Paused {
		paused = 1
	}

	var updatedAt int64
	if !sub.UpdatedAt.IsZero() {
		updatedAt = sub.UpdatedAt.Unix()
	}

	notifiedIDs := sub.NotifiedProductIDs
	if notifiedIDs == "" {
		notifiedIDs = "[]"
	}

	_, err := s.db.Exec(`
		INSERT INTO new_arrival_subscriptions (id, name, description, categories, models, chips, storages, memories,
			stock_statuses, max_price, min_price, keywords, bark_key, enabled, paused, created_at, updated_at, notified_product_ids)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sub.ID, sub.Name, sub.Description, string(categoriesJSON), string(modelsJSON), string(chipsJSON), string(storagesJSON), string(memoriesJSON),
		string(stockStatusesJSON), sub.MaxPrice, sub.MinPrice, string(keywordsJSON), sub.BarkKey, enabled, paused,
		sub.CreatedAt.Unix(), updatedAt, notifiedIDs)

	return err
}

// RemoveNewArrivalSubscription removes a new arrival subscription
func (s *SQLiteStore) RemoveNewArrivalSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM new_arrival_subscriptions WHERE id = ?", id)
	return err
}

// GetAllNewArrivalSubscriptions returns all new arrival subscriptions
func (s *SQLiteStore) GetAllNewArrivalSubscriptions() []*model.NewArrivalSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, description, categories, models, chips, storages, memories, stock_statuses,
		       max_price, min_price, keywords, bark_key, enabled, paused, notification_count,
		       last_notified_at, created_at, updated_at, notified_product_ids
		FROM new_arrival_subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return []*model.NewArrivalSubscription{}
	}
	defer rows.Close()

	var subs []*model.NewArrivalSubscription
	for rows.Next() {
		sub := &model.NewArrivalSubscription{}
		var created int64
		var description, categoriesStr, modelsStr, chipsStr, storagesStr, memoriesStr, stockStatusesStr sql.NullString
		var keywordsStr, notifiedIDsStr sql.NullString
		var barkKey sql.NullString
		var enabled, paused int
		var notificationCount int
		var maxPrice, minPrice sql.NullFloat64
		var lastNotifiedAt, updatedAt sql.NullInt64

		err := rows.Scan(&sub.ID, &sub.Name, &description, &categoriesStr, &modelsStr, &chipsStr, &storagesStr, &memoriesStr,
			&stockStatusesStr, &maxPrice, &minPrice, &keywordsStr, &barkKey, &enabled, &paused,
			&notificationCount, &lastNotifiedAt, &created, &updatedAt, &notifiedIDsStr)
		if err != nil {
			continue
		}

		sub.Description = description.String

		// Parse categories JSON using encoding/json
		// Need to unmarshal regardless of content - empty arrays are valid
		if categoriesStr.Valid && categoriesStr.String != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] categoriesStr.String = %q (len=%d)\n", categoriesStr.String, len(categoriesStr.String))
			fmt.Fprintf(os.Stderr, "[DEBUG] categoriesStr bytes = %v\n", []byte(categoriesStr.String))
			json.Unmarshal([]byte(categoriesStr.String), &sub.Categories)
			fmt.Fprintf(os.Stderr, "[DEBUG] After unmarshal, sub.Categories = %v\n", sub.Categories)
		}

		// Parse models JSON using encoding/json
		if modelsStr.Valid && modelsStr.String != "" {
			json.Unmarshal([]byte(modelsStr.String), &sub.Models)
		}

		// Parse chips JSON using encoding/json
		if chipsStr.Valid && chipsStr.String != "" {
			json.Unmarshal([]byte(chipsStr.String), &sub.Chips)
		}

		// Parse storages JSON using encoding/json
		if storagesStr.Valid && storagesStr.String != "" {
			json.Unmarshal([]byte(storagesStr.String), &sub.Storages)
		}

		// Parse memories JSON using encoding/json
		if memoriesStr.Valid && memoriesStr.String != "" {
			json.Unmarshal([]byte(memoriesStr.String), &sub.Memories)
		}

		// Parse stock_statuses JSON using encoding/json
		if stockStatusesStr.Valid && stockStatusesStr.String != "" {
			json.Unmarshal([]byte(stockStatusesStr.String), &sub.StockStatuses)
		}

		// Parse keywords JSON using encoding/json
		if keywordsStr.Valid && keywordsStr.String != "" {
			json.Unmarshal([]byte(keywordsStr.String), &sub.Keywords)
		}

		if barkKey.Valid {
			sub.BarkKey = barkKey.String
		}
		if notifiedIDsStr.Valid {
			sub.NotifiedProductIDs = notifiedIDsStr.String
		} else {
			sub.NotifiedProductIDs = "[]"
		}
		sub.Enabled = enabled == 1
		sub.Paused = paused == 1
		if maxPrice.Valid {
			sub.MaxPrice = maxPrice.Float64
		}
		if minPrice.Valid {
			sub.MinPrice = minPrice.Float64
		}
		sub.NotificationCount = notificationCount

		sub.CreatedAt = time.Unix(created, 0)
		subs = append(subs, sub)
	}

	return subs
}

// GetNewArrivalSubscriptionsByBarkKey returns subscriptions for a specific Bark Key
func (s *SQLiteStore) GetNewArrivalSubscriptionsByBarkKey(barkKey string) []*model.NewArrivalSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, description, categories, models, chips, storages, memories, stock_statuses,
		       max_price, min_price, keywords, bark_key, enabled, paused, notification_count,
		       last_notified_at, created_at, updated_at, notified_product_ids
		FROM new_arrival_subscriptions
		WHERE bark_key = ?
		ORDER BY created_at DESC
	`, barkKey)
	if err != nil {
		return []*model.NewArrivalSubscription{}
	}
	defer rows.Close()

	var subs []*model.NewArrivalSubscription
	for rows.Next() {
		sub := &model.NewArrivalSubscription{}
		var created int64
		var description, categoriesStr, modelsStr, chipsStr, storagesStr, memoriesStr, stockStatusesStr sql.NullString
		var keywordsStr, notifiedIDsStr sql.NullString
		var barkKeyVal sql.NullString
		var enabled, paused int
		var notificationCount int
		var maxPrice, minPrice sql.NullFloat64
		var lastNotifiedAt, updatedAt sql.NullInt64

		err := rows.Scan(&sub.ID, &sub.Name, &description, &categoriesStr, &modelsStr, &chipsStr, &storagesStr, &memoriesStr,
			&stockStatusesStr, &maxPrice, &minPrice, &keywordsStr, &barkKeyVal, &enabled, &paused,
			&notificationCount, &lastNotifiedAt, &created, &updatedAt, &notifiedIDsStr)
		if err != nil {
			continue
		}

		sub.Description = description.String

		// Parse categories JSON
		if categoriesStr.Valid && categoriesStr.String != "" {
			json.Unmarshal([]byte(categoriesStr.String), &sub.Categories)
		}

		// Parse models JSON
		if modelsStr.Valid && modelsStr.String != "" {
			json.Unmarshal([]byte(modelsStr.String), &sub.Models)
		}

		// Parse chips JSON
		if chipsStr.Valid && chipsStr.String != "" {
			json.Unmarshal([]byte(chipsStr.String), &sub.Chips)
		}

		// Parse storages JSON
		if storagesStr.Valid && storagesStr.String != "" {
			json.Unmarshal([]byte(storagesStr.String), &sub.Storages)
		}

		// Parse memories JSON
		if memoriesStr.Valid && memoriesStr.String != "" {
			json.Unmarshal([]byte(memoriesStr.String), &sub.Memories)
		}

		// Parse stock_statuses JSON
		if stockStatusesStr.Valid && stockStatusesStr.String != "" {
			json.Unmarshal([]byte(stockStatusesStr.String), &sub.StockStatuses)
		}

		// Parse keywords JSON
		if keywordsStr.Valid && keywordsStr.String != "" {
			json.Unmarshal([]byte(keywordsStr.String), &sub.Keywords)
		}

		if barkKeyVal.Valid {
			sub.BarkKey = barkKeyVal.String
		}
		if notifiedIDsStr.Valid {
			sub.NotifiedProductIDs = notifiedIDsStr.String
		} else {
			sub.NotifiedProductIDs = "[]"
		}
		sub.Enabled = enabled == 1
		sub.Paused = paused == 1
		if maxPrice.Valid {
			sub.MaxPrice = maxPrice.Float64
		}
		if minPrice.Valid {
			sub.MinPrice = minPrice.Float64
		}
		sub.NotificationCount = notificationCount

		sub.CreatedAt = time.Unix(created, 0)
		subs = append(subs, sub)
	}

	return subs
}

// GetNewArrivalSubscription returns a new arrival subscription by ID
func (s *SQLiteStore) GetNewArrivalSubscription(id string) (*model.NewArrivalSubscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub := &model.NewArrivalSubscription{}
	var created int64
	var description, categoriesStr, modelsStr, chipsStr, storagesStr, memoriesStr, stockStatusesStr sql.NullString
	var keywordsStr, notifiedIDsStr sql.NullString
	var barkKey sql.NullString
	var enabled, paused int
	var notificationCount int
	var maxPrice, minPrice sql.NullFloat64
	var lastNotifiedAt, updatedAt sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, name, description, categories, models, chips, storages, memories, stock_statuses,
		       max_price, min_price, keywords, bark_key, enabled, paused, notification_count,
		       last_notified_at, created_at, updated_at, notified_product_ids
		FROM new_arrival_subscriptions WHERE id = ?
	`, id).Scan(&sub.ID, &sub.Name, &description, &categoriesStr, &modelsStr, &chipsStr, &storagesStr, &memoriesStr,
		&stockStatusesStr, &maxPrice, &minPrice, &keywordsStr, &barkKey, &enabled, &paused,
		&notificationCount, &lastNotifiedAt, &created, &updatedAt, &notifiedIDsStr)

	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	sub.Description = description.String

	// Parse categories JSON using encoding/json
	if categoriesStr.Valid && categoriesStr.String != "" && categoriesStr.String != "[]" {
		json.Unmarshal([]byte(categoriesStr.String), &sub.Categories)
	}

	// Parse models JSON using encoding/json
	if modelsStr.Valid && modelsStr.String != "" && modelsStr.String != "[]" {
		json.Unmarshal([]byte(modelsStr.String), &sub.Models)
	}

	// Parse chips JSON using encoding/json
	if chipsStr.Valid && chipsStr.String != "" && chipsStr.String != "[]" {
		json.Unmarshal([]byte(chipsStr.String), &sub.Chips)
	}

	// Parse storages JSON using encoding/json
	if storagesStr.Valid && storagesStr.String != "" && storagesStr.String != "[]" {
		json.Unmarshal([]byte(storagesStr.String), &sub.Storages)
	}

	// Parse memories JSON using encoding/json
	if memoriesStr.Valid && memoriesStr.String != "" && memoriesStr.String != "[]" {
		json.Unmarshal([]byte(memoriesStr.String), &sub.Memories)
	}

	// Parse stock_statuses JSON using encoding/json
	if stockStatusesStr.Valid && stockStatusesStr.String != "" && stockStatusesStr.String != "[]" {
		json.Unmarshal([]byte(stockStatusesStr.String), &sub.StockStatuses)
	}

	// Parse keywords JSON using encoding/json
	if keywordsStr.Valid && keywordsStr.String != "" && keywordsStr.String != "[]" {
		json.Unmarshal([]byte(keywordsStr.String), &sub.Keywords)
	}

	if barkKey.Valid {
		sub.BarkKey = barkKey.String
	}
	if notifiedIDsStr.Valid {
		sub.NotifiedProductIDs = notifiedIDsStr.String
	} else {
		sub.NotifiedProductIDs = "[]"
	}
	sub.Enabled = enabled == 1
	sub.Paused = paused == 1
	sub.NotificationCount = notificationCount
	if maxPrice.Valid {
		sub.MaxPrice = maxPrice.Float64
	}
	if minPrice.Valid {
		sub.MinPrice = minPrice.Float64
	}
	if lastNotifiedAt.Valid {
		sub.LastNotifiedAt = time.Unix(lastNotifiedAt.Int64, 0)
	}
	sub.CreatedAt = time.Unix(created, 0)
	if updatedAt.Valid {
		sub.UpdatedAt = time.Unix(updatedAt.Int64, 0)
	}

	return sub, true
}

// UpdateNotifiedProductIDs adds a product ID to the notified list
func (s *SQLiteStore) UpdateNotifiedProductIDs(subscriptionID, productID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current notified_product_ids
	var currentIDs sql.NullString
	err := s.db.QueryRow("SELECT notified_product_ids FROM new_arrival_subscriptions WHERE id = ?", subscriptionID).Scan(&currentIDs)
	if err != nil {
		return err
	}

	// Parse existing IDs
	var ids []string
	if currentIDs.Valid && currentIDs.String != "" && currentIDs.String != "[]" {
		// Simple JSON parsing for array of strings
		trimmed := strings.Trim(currentIDs.String, "[]")
		if trimmed != "" {
			ids = strings.Split(trimmed, "\",\"")
			// Clean up quotes
			for i := range ids {
				ids[i] = strings.Trim(ids[i], "\"")
			}
		}
	}

	// Check if already notified
	for _, id := range ids {
		if id == productID {
			return nil // Already notified
		}
	}

	// Add new ID
	ids = append(ids, productID)

	// Build JSON array
	newIDs := "[]"
	if len(ids) > 0 {
		quotedIDs := make([]string, len(ids))
		for i, id := range ids {
			quotedIDs[i] = "\"" + id + "\""
		}
		newIDs = "[" + strings.Join(quotedIDs, ",") + "]"
	}

	// Update database
	_, err = s.db.Exec("UPDATE new_arrival_subscriptions SET notified_product_ids = ? WHERE id = ?", newIDs, subscriptionID)
	return err
}

// AddNotificationHistory adds a notification history record
func (s *SQLiteStore) AddNotificationHistory(history *model.NotificationHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO notification_history (id, subscription_id, product_id, product_name, product_category,
			product_price, product_image_url, product_specs, notification_type, status, error_message,
			bark_key, bark_key_masked, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, history.ID, history.SubscriptionID, history.ProductID, history.ProductName,
		history.ProductCategory, history.ProductPrice, history.ProductImageURL, history.ProductSpecs,
		history.NotificationType, history.Status, history.ErrorMessage, history.BarkKey, history.BarkKeyMasked,
		history.CreatedAt.Unix())

	return err
}

// GetNotificationHistory retrieves notification history with optional filters
func (s *SQLiteStore) GetNotificationHistory(subscriptionID string, barkKey string, limit, offset int) ([]*model.NotificationHistory, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build query with filters - always filter by bark_key for user isolation
	query := `SELECT id, subscription_id, product_id, product_name, product_category, product_price,
		product_image_url, product_specs, notification_type, status, error_message, bark_key, bark_key_masked,
		created_at, read_at FROM notification_history WHERE bark_key = ?`
	args := []interface{}{barkKey}

	if subscriptionID != "" {
		query += " AND subscription_id = ?"
		args = append(args, subscriptionID)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM notification_history WHERE bark_key = ?"
	countArgs := []interface{}{barkKey}
	if subscriptionID != "" {
		countQuery += " AND subscription_id = ?"
		countArgs = append(countArgs, subscriptionID)
	}
	var total int
	_ = s.db.QueryRow(countQuery, countArgs...).Scan(&total)

	// Add order and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return []*model.NotificationHistory{}, 0
	}
	defer rows.Close()

	var history []*model.NotificationHistory
	for rows.Next() {
		h := &model.NotificationHistory{}
		var created int64
		var readAt sql.NullInt64
		var barkKeyFull sql.NullString

		err := rows.Scan(&h.ID, &h.SubscriptionID, &h.ProductID, &h.ProductName, &h.ProductCategory,
			&h.ProductPrice, &h.ProductImageURL, &h.ProductSpecs, &h.NotificationType, &h.Status,
			&h.ErrorMessage, &barkKeyFull, &h.BarkKeyMasked, &created, &readAt)
		if err != nil {
			continue
		}

		h.CreatedAt = time.Unix(created, 0)
		if readAt.Valid {
			readTime := time.Unix(readAt.Int64, 0)
			h.ReadAt = &readTime
		}

		history = append(history, h)
	}

	return history, total
}

// MarkNotificationAsRead marks a notification as read
func (s *SQLiteStore) MarkNotificationAsRead(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE notification_history SET read_at = ? WHERE id = ?", time.Now().Unix(), id)
	return err
}

// GetUnreadNotificationCount returns the count of unread notifications
func (s *SQLiteStore) GetUnreadNotificationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM notification_history WHERE read_at IS NULL").Scan(&count)
	return count
}

// UpdateNewArrivalSubscription updates an existing subscription
func (s *SQLiteStore) UpdateNewArrivalSubscription(sub *model.NewArrivalSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use json.Marshal for proper JSON encoding
	categoriesJSON, _ := json.Marshal(sub.Categories)
	modelsJSON, _ := json.Marshal(sub.Models)
	chipsJSON, _ := json.Marshal(sub.Chips)
	storagesJSON, _ := json.Marshal(sub.Storages)
	memoriesJSON, _ := json.Marshal(sub.Memories)
	stockStatusesJSON, _ := json.Marshal(sub.StockStatuses)
	keywordsJSON, _ := json.Marshal(sub.Keywords)

	paused := 0
	if sub.Paused {
		paused = 1
	}

	enabled := 1
	if !sub.Enabled {
		enabled = 0
	}

	var updatedAt int64
	if !sub.UpdatedAt.IsZero() {
		updatedAt = sub.UpdatedAt.Unix()
	}

	_, err := s.db.Exec(`
		UPDATE new_arrival_subscriptions
		SET name = ?, description = ?, categories = ?, models = ?, chips = ?, storages = ?,
		    memories = ?, stock_statuses = ?, min_price = ?, max_price = ?,
		    keywords = ?, bark_key = ?, enabled = ?, paused = ?, updated_at = ?
		WHERE id = ?
	`, sub.Name, sub.Description, string(categoriesJSON), string(modelsJSON), string(chipsJSON), string(storagesJSON),
		string(memoriesJSON), string(stockStatusesJSON), sub.MinPrice, sub.MaxPrice,
		string(keywordsJSON), sub.BarkKey, enabled, paused, updatedAt, sub.ID)

	return err
}

// PauseSubscription pauses a subscription
func (s *SQLiteStore) PauseSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE new_arrival_subscriptions SET paused = 1 WHERE id = ?", id)
	return err
}

// ResumeSubscription resumes a paused subscription
func (s *SQLiteStore) ResumeSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE new_arrival_subscriptions SET paused = 0 WHERE id = ?", id)
	return err
}

// IncrementNotificationCount increments the notification count for a subscription
func (s *SQLiteStore) IncrementNotificationCount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE new_arrival_subscriptions
		SET notification_count = notification_count + 1, last_notified_at = ?
		WHERE id = ?
	`, time.Now().Unix(), id)

	return err
}

// GetScraperStatus returns the current scraper status
func (s *SQLiteStore) GetScraperStatus() *model.ScraperStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &model.ScraperStatus{}
	var lastTime, updatedAt sql.NullInt64
	var scrapeErr sql.NullString
	var productsScraped sql.NullInt64
	var duration sql.NullInt64

	err := s.db.QueryRow(`
		SELECT last_scrape_time, last_scrape_status, last_scrape_error,
			   products_scraped, duration_ms, updated_at
		FROM scraper_status WHERE id = 1
	`).Scan(&lastTime, &status.LastScrapeStatus, &scrapeErr,
		&productsScraped, &duration, &updatedAt)

	if err == sql.ErrNoRows {
		status.LastScrapeStatus = "never"
		return status
	}

	if lastTime.Valid {
		status.LastScrapeTime = time.Unix(lastTime.Int64, 0)
	}
	if scrapeErr.Valid {
		status.LastScrapeError = scrapeErr.String
	}
	if productsScraped.Valid {
		status.ProductsScraped = int(productsScraped.Int64)
	}
	if duration.Valid {
		status.Duration = duration.Int64
	}

	return status
}

// UpdateScraperStatus updates the scraper status
func (s *SQLiteStore) UpdateScraperStatus(status *model.ScraperStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastTime interface{}
	if !status.LastScrapeTime.IsZero() {
		lastTime = status.LastScrapeTime.Unix()
	} else {
		lastTime = nil
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO scraper_status
		(id, last_scrape_time, last_scrape_status, last_scrape_error, products_scraped, duration_ms, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`, lastTime, status.LastScrapeStatus, status.LastScrapeError,
		status.ProductsScraped, status.Duration, time.Now().Unix())

	return err
}
