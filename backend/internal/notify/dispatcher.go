package notify

import (
	"log"
	"sync"

	"apple-price/internal/model"
)

// Dispatcher handles notification dispatch for price changes
type Dispatcher struct {
	bark  *BarkService
	email *EmailService
	mu    sync.RWMutex
}

// NewDispatcher creates a new notification dispatcher
func NewDispatcher(bark *BarkService, email *EmailService) *Dispatcher {
	return &Dispatcher{
		bark:  bark,
		email: email,
	}
}

// NotifyPriceChange notifies subscribers of a price change
func (d *Dispatcher) NotifyPriceChange(product *model.Product, oldPrice, newPrice float64, subscriptions []*model.Subscription) error {
	// Read services under lock to avoid holding lock during goroutine execution
	d.mu.RLock()
	bark := d.bark
	email := d.email
	d.mu.RUnlock()

	if len(subscriptions) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(subscriptions)*2)

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

			// Send Email notification
			if s.Email != "" && email != nil && email.IsEnabled() {
				if err := email.SendPriceChangeEmail(
					s.Email,
					product.Name,
					oldPrice,
					newPrice,
					product.ProductURL,
				); err != nil {
					log.Printf("Email notification failed for %s: %v", s.ID, err)
					errChan <- err
				} else {
					log.Printf("Email notification sent to %s for %s (price: %.0f, target: %.0f)",
						s.Email, product.Name, newPrice, s.TargetPrice)
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
	// Read bark service under lock
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

		// Email notifications for stock changes are optional
		// to avoid spamming users
	}

	return nil
}

// GetBarkService returns the Bark service
func (d *Dispatcher) GetBarkService() *BarkService {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.bark
}

// GetEmailService returns the email service
func (d *Dispatcher) GetEmailService() *EmailService {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.email
}

// SetBarkService sets the Bark service
func (d *Dispatcher) SetBarkService(bark *BarkService) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.bark = bark
}

// SetEmailService sets the email service
func (d *Dispatcher) SetEmailService(email *EmailService) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.email = email
}
