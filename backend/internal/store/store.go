package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"apple-price/internal/model"
)

const (
	maxHistoryPerProduct = 100
)

// Store manages in-memory product data with JSON persistence
type Store struct {
	mu                sync.RWMutex
	products          map[string]*model.Product
	history           map[string][]model.PriceHistory
	prevPrices        map[string]float64
	subscriptions     map[string]*model.Subscription
	subscriptionsByProduct map[string][]string // productID -> subscriptionIDs
	newArrivalSubscriptions map[string]*model.NewArrivalSubscription
	notificationHistory    []*model.NotificationHistory
	dataDir           string
	lastScrapeTime    time.Time
	scraperStatus     *model.ScraperStatus
}

// New creates a new Store instance
func New(dataDir string) (*Store, error) {
	s := &Store{
		products:                 make(map[string]*model.Product),
		history:                  make(map[string][]model.PriceHistory),
		prevPrices:               make(map[string]float64),
		subscriptions:            make(map[string]*model.Subscription),
		subscriptionsByProduct:   make(map[string][]string),
		newArrivalSubscriptions:  make(map[string]*model.NewArrivalSubscription),
		notificationHistory:      make([]*model.NotificationHistory, 0),
		dataDir:                  dataDir,
	}

	// Create data directory if not exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Load existing data
	if err := s.Load(); err != nil {
		// Don't fail on first run, just log
		fmt.Printf("Warning: failed to load data: %v\n", err)
	}

	return s, nil
}

// Load loads data from JSON files
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load products
	productsFile := filepath.Join(s.dataDir, "products.json")
	if data, err := os.ReadFile(productsFile); err == nil {
		var products []*model.Product
		if err := json.Unmarshal(data, &products); err != nil {
			return fmt.Errorf("failed to unmarshal products: %w", err)
		}
		for _, p := range products {
			s.products[p.ID] = p
		}
	}

	// Load history
	historyFile := filepath.Join(s.dataDir, "history.json")
	if data, err := os.ReadFile(historyFile); err == nil {
		var history map[string][]model.PriceHistory
		if err := json.Unmarshal(data, &history); err != nil {
			return fmt.Errorf("failed to unmarshal history: %w", err)
		}
		s.history = history
	}

	// Load subscriptions
	subsFile := filepath.Join(s.dataDir, "subscriptions.json")
	if data, err := os.ReadFile(subsFile); err == nil {
		var subs map[string]*model.Subscription
		if err := json.Unmarshal(data, &subs); err != nil {
			return fmt.Errorf("failed to unmarshal subscriptions: %w", err)
		}
		s.subscriptions = subs
		// Rebuild product index
		for id, sub := range subs {
			s.subscriptionsByProduct[sub.ProductID] = append(
				s.subscriptionsByProduct[sub.ProductID],
				id,
			)
		}
	}

	// Load notification history
	notifHistoryFile := filepath.Join(s.dataDir, "notification_history.json")
	if data, err := os.ReadFile(notifHistoryFile); err == nil {
		var notifHistory []*model.NotificationHistory
		if err := json.Unmarshal(data, &notifHistory); err != nil {
			return fmt.Errorf("failed to unmarshal notification history: %w", err)
		}
		s.notificationHistory = notifHistory
	}

	return nil
}

// Save saves data to JSON files
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Save products
	products := make([]*model.Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	productsData, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal products: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, "products.json"), productsData, 0644); err != nil {
		return fmt.Errorf("failed to write products: %w", err)
	}

	// Save history
	historyData, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, "history.json"), historyData, 0644); err != nil {
		return fmt.Errorf("failed to write history: %w", err)
	}

	// Save subscriptions
	subsData, err := json.MarshalIndent(s.subscriptions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscriptions: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, "subscriptions.json"), subsData, 0644); err != nil {
		return fmt.Errorf("failed to write subscriptions: %w", err)
	}

	// Save notification history
	notifHistoryData, err := json.MarshalIndent(s.notificationHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notification history: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, "notification_history.json"), notifHistoryData, 0644); err != nil {
		return fmt.Errorf("failed to write notification history: %w", err)
	}

	return nil
}

// GetAllProducts returns all products
func (s *Store) GetAllProducts() []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	products := make([]*model.Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	return products
}

// GetProduct returns a product by ID
func (s *Store) GetProduct(id string) (*model.Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.products[id]
	return p, ok
}

// GetProductsByCategory returns products filtered by category
func (s *Store) GetProductsByCategory(category string) []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var products []*model.Product
	for _, p := range s.products {
		if p.Category == category {
			products = append(products, p)
		}
	}
	return products
}

// GetProductsByRegion returns products filtered by region
func (s *Store) GetProductsByRegion(region string) []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var products []*model.Product
	for _, p := range s.products {
		if p.Region == region {
			products = append(products, p)
		}
	}
	return products
}

// UpsertProduct adds or updates a product, returns true if price changed
func (s *Store) UpsertProduct(product *model.Product) (priceChanged bool, oldPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	existing, exists := s.products[product.ID]
	if exists {
		// Check for price change
		if existing.Price != product.Price {
			priceChanged = true
			oldPrice = existing.Price

			// Add to history
			s.history[product.ID] = append(s.history[product.ID], model.PriceHistory{
				ProductID: product.ID,
				Price:     existing.Price,
				Timestamp: now,
				Discount:  existing.Discount,
			})

			// Trim history if too long
			if len(s.history[product.ID]) > maxHistoryPerProduct {
				s.history[product.ID] = s.history[product.ID][1:]
			}
		}

		// Update created_at to preserve original creation time
		product.CreatedAt = existing.CreatedAt
	} else {
		product.CreatedAt = now

		// Initialize history with current price for new products
		s.history[product.ID] = []model.PriceHistory{
			{
				ProductID: product.ID,
				Price:     product.Price,
				Timestamp: now,
				Discount:  product.Discount,
			},
		}
	}

	product.UpdatedAt = now

	// Calculate value score based on discount and history
	product.ValueScore = s.calculateValueScore(product, s.history[product.ID], now)
	s.updatePriceStats(product, now)

	s.products[product.ID] = product

	return priceChanged, oldPrice
}

// calculateValueScore computes a 0-100 value score based on discount and price history
func (s *Store) calculateValueScore(product *model.Product, history []model.PriceHistory, now time.Time) float64 {
	score := 50.0 // Base score

	// Discount score: 0-30 points
	if product.Discount >= 15 {
		score += 30
	} else if product.Discount >= 12 {
		score += 25
	} else if product.Discount >= 10 {
		score += 20
	} else if product.Discount >= 8 {
		score += 15
	} else if product.Discount >= 5 {
		score += 10
	} else {
		score += product.Discount * 2 // Less than 5% gets proportional score
	}

	// Price trend score: 0-25 points
	if len(history) >= 2 {
		firstPrice := history[0].Price
		lastPrice := history[len(history)-1].Price
		change := (lastPrice - firstPrice) / firstPrice

		if change < -0.02 { // Price dropped >2%
			score += 25
		} else if change < -0.01 { // Price dropped >1%
			score += 20
		} else if change < 0 { // Price dropped
			score += 15
		} else if change > 0.02 { // Price rose >2%
			score += 0
		} else {
			score += 10 // Stable
		}
	}

	// Stock status score: 0-15 points
	if product.StockStatus == "available" {
		score += 15
	} else if product.StockStatus == "limited" {
		score += 10
	}
	// sold_out gets 0 points

	// Price position score: 0-20 points (current price vs historical range)
	if len(history) >= 2 {
		minPrice := history[0].Price
		maxPrice := history[0].Price
		for _, h := range history {
			if h.Price < minPrice {
				minPrice = h.Price
			}
			if h.Price > maxPrice {
				maxPrice = h.Price
			}
		}

		if maxPrice > minPrice {
			position := (product.Price - minPrice) / (maxPrice - minPrice)
			if position <= 0.1 {
				score += 20 // Near historical low
			} else if position <= 0.3 {
				score += 15
			} else if position <= 0.5 {
				score += 10
			} else if position <= 0.7 {
				score += 5
			}
			// Near historical high gets 0 points
		} else {
			score += 10 // No price variation
		}
	}

	// Age score: 0-10 points (newer listings get higher score)
	daysSinceCreation := now.Sub(product.CreatedAt).Hours() / 24
	if daysSinceCreation <= 7 {
		score += 10
	} else if daysSinceCreation <= 30 {
		score += 7
	} else if daysSinceCreation <= 90 {
		score += 3
	}

	// Clamp score to 0-100
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

// updatePriceStats updates lowest_price, highest_price, and price_trend
func (s *Store) updatePriceStats(product *model.Product, now time.Time) {
	history := s.history[product.ID]
	if len(history) == 0 {
		return
	}

	minPrice := history[0].Price
	maxPrice := history[0].Price
	for _, h := range history {
		if h.Price < minPrice {
			minPrice = h.Price
		}
		if h.Price > maxPrice {
			maxPrice = h.Price
		}
	}

	product.LowestPrice = minPrice
	product.HighestPrice = maxPrice

	// Determine trend
	if len(history) >= 3 {
		recent := history[len(history)-3:]
		startPrice := recent[0].Price
		endPrice := recent[2].Price
		change := (endPrice - startPrice) / startPrice

		if change < -0.02 {
			product.PriceTrend = "falling"
		} else if change > 0.02 {
			product.PriceTrend = "rising"
		} else {
			product.PriceTrend = "stable"
		}
	}
}

// GetPriceHistory returns price history for a product
func (s *Store) GetPriceHistory(productID string) []model.PriceHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.history[productID]
}

// GetCategories returns all unique categories
func (s *Store) GetCategories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categoryMap := make(map[string]bool)
	for _, p := range s.products {
		categoryMap[p.Category] = true
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}
	return categories
}

// AddSubscription adds a new subscription
func (s *Store) AddSubscription(sub *model.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscriptions[sub.ID] = sub
	s.subscriptionsByProduct[sub.ProductID] = append(
		s.subscriptionsByProduct[sub.ProductID],
		sub.ID,
	)

	return nil
}

// RemoveSubscription removes a subscription
func (s *Store) RemoveSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription not found")
	}

	// Remove from product index
	productSubs := s.subscriptionsByProduct[sub.ProductID]
	for i, sid := range productSubs {
		if sid == id {
			s.subscriptionsByProduct[sub.ProductID] = append(
				productSubs[:i],
				productSubs[i+1:]...,
			)
			break
		}
	}

	delete(s.subscriptions, id)
	return nil
}

// DeleteProductsByRegion deletes all products from a specific region
func (s *Store) DeleteProductsByRegion(region string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, p := range s.products {
		if p.Region == region {
			delete(s.products, id)
			count++
		}
	}
	return count, nil
}

// GetSubscriptionsByProduct returns all subscriptions for a product
func (s *Store) GetSubscriptionsByProduct(productID string) []*model.Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subIDs := s.subscriptionsByProduct[productID]
	subs := make([]*model.Subscription, 0, len(subIDs))
	for _, id := range subIDs {
		if sub, ok := s.subscriptions[id]; ok {
			subs = append(subs, sub)
		}
	}
	return subs
}

// GetAllSubscriptions returns all subscriptions
func (s *Store) GetAllSubscriptions() []*model.Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subs := make([]*model.Subscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// UpdateLastScrapeTime updates the last scrape timestamp
func (s *Store) UpdateLastScrapeTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastScrapeTime = t
}

// GetLastScrapeTime returns the last scrape timestamp
func (s *Store) GetLastScrapeTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastScrapeTime
}

// GetStats returns system statistics
func (s *Store) GetStats() *model.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &model.Stats{
		TotalProducts:       len(s.products),
		Categories:          make(map[string]int),
		TotalSubscriptions:  len(s.subscriptions),
		LastScrapeTime:      s.lastScrapeTime,
		AvailableProducts:   0,
	}

	for _, p := range s.products {
		stats.Categories[p.Category]++
		if p.StockStatus == "available" {
			stats.AvailableProducts++
		}
	}

	return stats
}

// AddNewArrivalSubscription adds a new arrival subscription
func (s *Store) AddNewArrivalSubscription(sub *model.NewArrivalSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		s.newArrivalSubscriptions = make(map[string]*model.NewArrivalSubscription)
	}

	s.newArrivalSubscriptions[sub.ID] = sub
	return nil
}

// RemoveNewArrivalSubscription removes a new arrival subscription
func (s *Store) RemoveNewArrivalSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		return fmt.Errorf("new arrival subscription not found")
	}

	if _, exists := s.newArrivalSubscriptions[id]; !exists {
		return fmt.Errorf("new arrival subscription not found")
	}

	delete(s.newArrivalSubscriptions, id)
	return nil
}

// GetAllNewArrivalSubscriptions returns all new arrival subscriptions
func (s *Store) GetAllNewArrivalSubscriptions() []*model.NewArrivalSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.newArrivalSubscriptions == nil {
		return nil
	}

	subs := make([]*model.NewArrivalSubscription, 0, len(s.newArrivalSubscriptions))
	for _, sub := range s.newArrivalSubscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// GetNewArrivalSubscriptionsByBarkKey returns subscriptions for a specific Bark Key
func (s *Store) GetNewArrivalSubscriptionsByBarkKey(barkKey string) []*model.NewArrivalSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.newArrivalSubscriptions == nil {
		return nil
	}

	subs := make([]*model.NewArrivalSubscription, 0)
	for _, sub := range s.newArrivalSubscriptions {
		if sub.BarkKey == barkKey {
			subs = append(subs, sub)
		}
	}
	return subs
}

// GetNewArrivalSubscription returns a new arrival subscription by ID
func (s *Store) GetNewArrivalSubscription(id string) (*model.NewArrivalSubscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.newArrivalSubscriptions == nil {
		return nil, false
	}

	sub, ok := s.newArrivalSubscriptions[id]
	return sub, ok
}

// UpdateNotifiedProductIDs adds a product ID to the notified list
func (s *Store) UpdateNotifiedProductIDs(subscriptionID, productID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub, exists := s.newArrivalSubscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("new arrival subscription not found")
	}

	// Parse existing IDs
	var ids []string
	if sub.NotifiedProductIDs != "" && sub.NotifiedProductIDs != "[]" {
		// Simple JSON parsing
		trimmed := sub.NotifiedProductIDs[1 : len(sub.NotifiedProductIDs)-1]
		if trimmed != "" {
			// Split by comma and clean quotes
			parts := strings.Split(trimmed, ",")
			for _, part := range parts {
				cleaned := strings.TrimSpace(part)
				cleaned = strings.Trim(cleaned, `"`)
				if cleaned != "" {
					ids = append(ids, cleaned)
				}
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
	if len(ids) == 0 {
		sub.NotifiedProductIDs = "[]"
	} else {
		quotedIDs := make([]string, len(ids))
		for i, id := range ids {
			quotedIDs[i] = "\"" + id + "\""
		}
		sub.NotifiedProductIDs = "[" + strings.Join(quotedIDs, ",") + "]"
	}

	return nil
}

// AddNotificationHistory adds a notification history record
func (s *Store) AddNotificationHistory(history *model.NotificationHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notificationHistory = append(s.notificationHistory, history)
	return nil
}

// GetNotificationHistory returns notification history with pagination
func (s *Store) GetNotificationHistory(subscriptionID string, barkKey string, limit, offset int) ([]*model.NotificationHistory, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter by bark_key first (user isolation), then by subscription ID if provided
	var filtered []*model.NotificationHistory
	for _, h := range s.notificationHistory {
		if h.BarkKey == barkKey {
			if subscriptionID == "" || h.SubscriptionID == subscriptionID {
				filtered = append(filtered, h)
			}
		}
	}

	total := len(filtered)

	// Apply pagination
	if offset >= total {
		return nil, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total
}

// MarkNotificationAsRead marks a notification as read
func (s *Store) MarkNotificationAsRead(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, h := range s.notificationHistory {
		if h.ID == id {
			h.ReadAt = &now
			return nil
		}
	}

	return fmt.Errorf("notification not found")
}

// GetUnreadNotificationCount returns the count of unread notifications
func (s *Store) GetUnreadNotificationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, h := range s.notificationHistory {
		if h.ReadAt == nil {
			count++
		}
	}
	return count
}

// UpdateNewArrivalSubscription updates an existing subscription
func (s *Store) UpdateNewArrivalSubscription(sub *model.NewArrivalSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		return fmt.Errorf("new arrival subscription not found")
	}

	if _, exists := s.newArrivalSubscriptions[sub.ID]; !exists {
		return fmt.Errorf("new arrival subscription not found")
	}

	s.newArrivalSubscriptions[sub.ID] = sub
	return nil
}

// PauseSubscription pauses a subscription
func (s *Store) PauseSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub, exists := s.newArrivalSubscriptions[id]
	if !exists {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub.Paused = true
	return nil
}

// ResumeSubscription resumes a paused subscription
func (s *Store) ResumeSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub, exists := s.newArrivalSubscriptions[id]
	if !exists {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub.Paused = false
	return nil
}

// IncrementNotificationCount increments the notification count for a subscription
func (s *Store) IncrementNotificationCount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.newArrivalSubscriptions == nil {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub, exists := s.newArrivalSubscriptions[id]
	if !exists {
		return fmt.Errorf("new arrival subscription not found")
	}

	sub.NotificationCount++
	sub.LastNotifiedAt = time.Now()
	return nil
}

// GetScraperStatus returns the current scraper status (in-memory for JSON store)
func (s *Store) GetScraperStatus() *model.ScraperStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.scraperStatus == nil {
		return &model.ScraperStatus{
			LastScrapeStatus: "never",
		}
	}
	return s.scraperStatus
}

// UpdateScraperStatus updates the scraper status (in-memory for JSON store)
func (s *Store) UpdateScraperStatus(status *model.ScraperStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scraperStatus = status
	return nil
}
