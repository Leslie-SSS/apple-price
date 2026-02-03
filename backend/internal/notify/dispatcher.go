package notify

import (
	"encoding/json"
	"log"
	"sync"

	"apple-price/internal/model"
)

// StoreInterface for updating notified product IDs
type StoreInterface interface {
	UpdateNotifiedProductIDs(subscriptionID, productID string) error
}

// Dispatcher handles notification dispatch for price changes
type Dispatcher struct {
	bark  *BarkService
	store StoreInterface
	mu    sync.RWMutex
}

// NewDispatcher creates a new notification dispatcher
func NewDispatcher(bark *BarkService) *Dispatcher {
	return &Dispatcher{
		bark: bark,
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

	for _, sub := range subscriptions {
		// Skip disabled subscriptions
		if !sub.Enabled {
			continue
		}

		// Check if product has already been notified
		var notifiedIDs []string
		if sub.NotifiedProductIDs != "" {
			if err := json.Unmarshal([]byte(sub.NotifiedProductIDs), &notifiedIDs); err == nil {
				for _, id := range notifiedIDs {
					if id == product.ID {
						// Already notified, skip
						continue
					}
				}
			}
		}

		// Check if product matches subscription criteria
		if !d.matchesSubscription(product, sub) {
			continue
		}

		// Send Bark notification
		if sub.BarkKey != "" && bark != nil {
			if err := bark.SendNewArrivalNotification(
				sub.BarkKey,
				product.Name,
				product.Price,
				product.Category,
				product.ProductURL,
			); err != nil {
				log.Printf("Bark new arrival notification failed for %s: %v", sub.ID, err)
				continue
			}

			log.Printf("New arrival notification sent to %s for product %s", sub.Name, product.Name)

			// Update store with notified product ID
			if store != nil {
				if err := store.UpdateNotifiedProductIDs(sub.ID, product.ID); err != nil {
					log.Printf("Failed to update notified_product_ids for %s: %v", sub.ID, err)
				}
			}
		}
	}

	return nil
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
			if contains(product.Name, kw) {
				keywordMatch = true
				break
			}
		}
		if !keywordMatch {
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
