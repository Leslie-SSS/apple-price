package store

import (
	"time"

	"apple-price/internal/model"
)

// StoreInterface defines the complete interface for product storage
// Both JSON Store and SQLite Store implement this interface
type StoreInterface interface {
	// Product operations
	GetAllProducts() []*model.Product
	GetProduct(id string) (*model.Product, bool)
	GetProductsByCategory(category string) []*model.Product
	GetProductsByRegion(region string) []*model.Product
	UpsertProduct(product *model.Product) (priceChanged bool, oldPrice float64)

	// Price history operations
	GetPriceHistory(productID string) []model.PriceHistory

	// Category operations
	GetCategories() []string

	// Subscription operations
	AddSubscription(sub *model.Subscription) error
	RemoveSubscription(id string) error
	GetSubscriptionsByProduct(productID string) []*model.Subscription
	GetAllSubscriptions() []*model.Subscription

	// New arrival subscription operations
	AddNewArrivalSubscription(sub *model.NewArrivalSubscription) error
	RemoveNewArrivalSubscription(id string) error
	GetAllNewArrivalSubscriptions() []*model.NewArrivalSubscription
	GetNewArrivalSubscription(id string) (*model.NewArrivalSubscription, bool)

	// Statistics operations
	GetStats() *model.Stats

	// Admin operations
	DeleteProductsByRegion(region string) (int, error)

	// Scraping metadata operations
	UpdateLastScrapeTime(t time.Time)
	GetLastScrapeTime() time.Time

	// Persistence
	Save() error
}
