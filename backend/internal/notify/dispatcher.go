package notify

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"apple-price/internal/model"
)

// StoreInterface for updating notified product IDs
type StoreInterface interface {
	UpdateNotifiedProductIDs(subscriptionID, productID string) error
	AddNotificationHistory(history *model.NotificationHistory) error
	IncrementNotificationCount(id string) error
}

// Dispatcher handles notification dispatch for price changes
type Dispatcher struct {
	bark  *BarkService
	store StoreInterface
	mu    sync.RWMutex
}

// NewDispatcher creates a new notification dispatcher
func NewDispatcher(bark *BarkService, store StoreInterface) *Dispatcher {
	return &Dispatcher{
		bark:  bark,
		store: store,
	}
}

// SetStore sets the store for updating notified product IDs
func (d *Dispatcher) SetStore(store StoreInterface) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store = store
}

// NotifyPriceChange notifies subscribers of a price change
func (d *Dispatcher) NotifyPriceChange(product *model.Product, oldPrice, newPrice float64, subscriptions []*model.Subscription) error {
	d.mu.RLock()
	bark := d.bark
	d.mu.RUnlock()

	if len(subscriptions) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(subscriptions))

	for _, sub := range subscriptions {
		wg.Add(1)
		go func(s *model.Subscription) {
			defer wg.Done()

			// Check target price condition (断层领先: 价格到达目标价才通知)
			if s.TargetPrice > 0 && newPrice > s.TargetPrice {
				// Price hasn't reached target yet, skip notification
				return
			}

			// Send Bark notification
			if s.BarkKey != "" && bark != nil {
				if err := bark.SendPriceChangeNotification(
					s.BarkKey,
					product.Name,
					oldPrice,
					newPrice,
					product.ProductURL,
				); err != nil {
					log.Printf("Bark notification failed for %s: %v", s.ID, err)
					errChan <- err
				} else {
					log.Printf("Bark notification sent to %s for %s (price: %.0f, target: %.0f)",
						s.BarkKey, product.Name, newPrice, s.TargetPrice)
				}
			}
		}(sub)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		log.Printf("Notification dispatch completed with %d errors", len(errors))
	}

	return nil
}

// NotifyStockChange notifies subscribers of stock status change
func (d *Dispatcher) NotifyStockChange(product *model.Product, oldStatus, newStatus string, subscriptions []*model.Subscription) error {
	d.mu.RLock()
	bark := d.bark
	d.mu.RUnlock()

	for _, sub := range subscriptions {
		// Send Bark notification
		if sub.BarkKey != "" && bark != nil {
			if err := bark.SendStockNotification(
				sub.BarkKey,
				product.Name,
				newStatus,
				product.ProductURL,
			); err != nil {
				log.Printf("Bark stock notification failed for %s: %v", sub.ID, err)
			}
		}
	}

	return nil
}

// NotifyNewArrival notifies subscribers when new products arrive
func (d *Dispatcher) NotifyNewArrival(product *model.Product, subscriptions []*model.NewArrivalSubscription) error {
	d.mu.RLock()
	bark := d.bark
	store := d.store
	d.mu.RUnlock()

	if len(subscriptions) == 0 {
		return nil
	}

	if store == nil {
		return nil
	}

	for _, sub := range subscriptions {
		// Skip disabled or paused subscriptions
		if !sub.Enabled || sub.Paused {
			continue
		}

		// Skip if no Bark Key configured for this subscription
		if sub.BarkKey == "" {
			continue
		}

		// Check if product has already been notified
		var notifiedIDs []string
		if sub.NotifiedProductIDs != "" {
			if err := json.Unmarshal([]byte(sub.NotifiedProductIDs), &notifiedIDs); err == nil {
				alreadyNotified := false
				for _, id := range notifiedIDs {
					if id == product.ID {
						alreadyNotified = true
						break
					}
				}
				if alreadyNotified {
					continue // Skip to next subscription
				}
			}
		}

		// Check if product matches subscription criteria
		if !d.matchesSubscription(product, sub) {
			continue
		}

		// Send Bark notification using subscription's Bark Key
		if bark != nil {
			var err error
			// Use enhanced notification with specs
			if err = bark.SendNewArrivalNotificationEnhanced(
				sub.BarkKey,
				product.Name,
				product.Category,
				product.Price,
				product.Discount,
				product.ImageURL,
				product.ProductURL,
				product.SpecsDetail,
			); err != nil {
				log.Printf("Bark new arrival notification failed for %s: %v", sub.ID, err)

				// Record failed notification history
				d.recordNotificationHistory(store, sub.ID, sub.BarkKey, product, "failed", err.Error())
				continue
			}

			log.Printf("New arrival notification sent for subscription %s, product %s", sub.Name, product.Name)

			// Record successful notification history
			d.recordNotificationHistory(store, sub.ID, sub.BarkKey, product, "sent", "")

			// Update notified product IDs and increment count
			if err := store.UpdateNotifiedProductIDs(sub.ID, product.ID); err != nil {
				log.Printf("Failed to update notified_product_ids for %s: %v", sub.ID, err)
			}
			if err := store.IncrementNotificationCount(sub.ID); err != nil {
				log.Printf("Failed to increment notification count for %s: %v", sub.ID, err)
			}
		}
	}

	return nil
}

// recordNotificationHistory records a notification in history
func (d *Dispatcher) recordNotificationHistory(store StoreInterface, subscriptionID string, barkKey string, product *model.Product, status, errorMsg string) {
	// Mask the Bark key for privacy
	maskedKey := ""
	if len(barkKey) > 0 {
		maskedKey = barkKey[:4] + "****" + barkKey[len(barkKey)-4:]
	}

	history := &model.NotificationHistory{
		ID:              generateHistoryID(),
		SubscriptionID:  subscriptionID,
		ProductID:       product.ID,
		ProductName:     product.Name,
		ProductCategory: product.Category,
		ProductPrice:    product.Price,
		ProductImageURL: product.ImageURL,
		ProductSpecs:    product.SpecsDetail,
		NotificationType: "new_arrival",
		Status:          status,
		ErrorMessage:    errorMsg,
		BarkKey:         barkKey,
		BarkKeyMasked:   maskedKey,
		CreatedAt:       time.Now(),
	}

	if err := store.AddNotificationHistory(history); err != nil {
		log.Printf("Failed to record notification history: %v", err)
	}
}

// generateHistoryID generates a unique ID for notification history
func generateHistoryID() string {
	return fmt.Sprintf("nh-%d", time.Now().UnixNano())
}

// matchesSubscription checks if a product matches the subscription criteria
func (d *Dispatcher) matchesSubscription(product *model.Product, sub *model.NewArrivalSubscription) bool {
	// Check category filter
	if len(sub.Categories) > 0 {
		categoryMatch := false
		for _, cat := range sub.Categories {
			if cat == product.Category {
				categoryMatch = true
				break
			}
		}
		if !categoryMatch {
			return false
		}
	}

	// Check model filter
	if len(sub.Models) > 0 {
		modelMatch := false
		productName := product.Name
		for _, model := range sub.Models {
			if contains(productName, model) {
				modelMatch = true
				break
			}
		}
		if !modelMatch {
			return false
		}
	}

	// Check price range
	if sub.MinPrice > 0 && product.Price < sub.MinPrice {
		return false
	}
	if sub.MaxPrice > 0 && product.Price > sub.MaxPrice {
		return false
	}

	// Check keywords
	if len(sub.Keywords) > 0 {
		keywordMatch := false
		for _, kw := range sub.Keywords {
			if contains(product.Name, kw) || contains(product.Specs, kw) {
				keywordMatch = true
				break
			}
		}
		if !keywordMatch {
			return false
		}
	}

	// Check chip filter
	if len(sub.Chips) > 0 {
		chipMatch := false
		productSpecsLower := toLower(product.Specs + " " + product.Name)
		for _, chip := range sub.Chips {
			if containsIgnoreCase(productSpecsLower, toLower(chip)) {
				chipMatch = true
				break
			}
		}
		if !chipMatch {
			return false
		}
	}

	// Check storage filter
	if len(sub.Storages) > 0 {
		storageMatch := false
		productSpecsLower := toLower(product.Specs)
		for _, storage := range sub.Storages {
			if containsIgnoreCase(productSpecsLower, toLower(storage)) {
				storageMatch = true
				break
			}
		}
		if !storageMatch {
			return false
		}
	}

	// Check memory filter
	if len(sub.Memories) > 0 {
		memoryMatch := false
		productSpecsLower := toLower(product.Specs)
		for _, memory := range sub.Memories {
			if containsIgnoreCase(productSpecsLower, toLower(memory)) {
				memoryMatch = true
				break
			}
		}
		if !memoryMatch {
			return false
		}
	}

	// Check stock status filter
	if len(sub.StockStatuses) > 0 {
		stockMatch := false
		for _, status := range sub.StockStatuses {
			if product.StockStatus == status {
				stockMatch = true
				break
			}
		}
		if !stockMatch {
			return false
		}
	}

	return true
}

// contains is a case-insensitive substring check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsIgnoreCase(s, substr)))
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// GetBarkService returns the Bark service
func (d *Dispatcher) GetBarkService() *BarkService {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.bark
}
