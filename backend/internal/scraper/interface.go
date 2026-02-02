package scraper

import (
	"apple-price/internal/model"
)

// Scraper defines the interface for product scrapers
type Scraper interface {
	ScrapeAll() ([]*model.Product, error)
}

// Ensure AppleScraper implements the interface
var _ Scraper = (*AppleScraper)(nil)
