package scraper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"apple-price/internal/model"
)

// DetailScraper handles asynchronous detail fetching with retry logic
type DetailScraper struct {
	scraper      *AppleScraper
	store        StoreInterface
	queue        chan *model.Product
	workers      int
	retryMax     int
	retryDelay   time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
	isRunning    bool
	mu           sync.RWMutex
	stats        DetailStats
}

// DetailStats tracks scraping statistics
type DetailStats struct {
	TotalQueued     int64
	TotalProcessed  int64
	TotalSuccess    int64
	TotalFailed     int64
	TotalRetries    int64
}

// NewDetailScraper creates a new asynchronous detail scraper
func NewDetailScraper(scraper *AppleScraper, store StoreInterface, workers int) *DetailScraper {
	return &DetailScraper{
		scraper:    scraper,
		store:      store,
		queue:      make(chan *model.Product, 1000),
		workers:    workers,
		retryMax:   3,
		retryDelay: 2 * time.Second,
		stopCh:     make(chan struct{}),
		stats:      DetailStats{},
	}
}

// Start begins processing the detail queue
func (d *DetailScraper) Start() {
	d.mu.Lock()
	if d.isRunning {
		d.mu.Unlock()
		return
	}
	d.isRunning = true
	d.mu.Unlock()

	log.Printf("[DetailScraper] Starting with %d workers", d.workers)

	d.wg.Add(d.workers)
	for i := 0; i < d.workers; i++ {
		go d.worker(i)
	}

	// Start stats reporter
	go d.statsReporter()
}

// Stop gracefully stops the detail scraper (idempotent)
func (d *DetailScraper) Stop() {
	d.mu.Lock()
	if !d.isRunning {
		d.mu.Unlock()
		return
	}
	d.isRunning = false
	d.mu.Unlock()

	// Idempotent channel close using sync.Once pattern
	select {
	case <-d.stopCh:
		// Already closed
	default:
		close(d.stopCh)
	}

	select {
	case _, ok := <-d.queue:
		if ok {
			close(d.queue)
		}
	default:
		close(d.queue)
	}

	d.wg.Wait()

	log.Printf("[DetailScraper] Stopped. Stats: Queued=%d, Processed=%d, Success=%d, Failed=%d, Retries=%d",
		d.stats.TotalQueued, d.stats.TotalProcessed, d.stats.TotalSuccess, d.stats.TotalFailed, d.stats.TotalRetries)
}

// Enqueue adds products to the detail queue
func (d *DetailScraper) Enqueue(products []*model.Product) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	count := 0
	for _, p := range products {
		// Skip if already has description
		if p.Description != "" {
			continue
		}
		// Skip if no product URL
		if p.ProductURL == "" {
			continue
		}
		// Skip HK products - they don't have meta descriptions on detail pages
		if p.Region == "hk" {
			continue
		}

		select {
		case d.queue <- p:
			d.stats.TotalQueued++
			count++
		default:
			// Queue full, skip this product
			log.Printf("[DetailScraper] Queue full, skipping %s", p.ID)
		}
	}

	if count > 0 {
		log.Printf("[DetailScraper] Enqueued %d products for detail fetching", count)
	}
	return count
}

// EnqueueSingle adds a single product to the queue
func (d *DetailScraper) EnqueueSingle(product *model.Product) bool {
	if product.Description != "" || product.ProductURL == "" {
		return false
	}

	select {
	case d.queue <- product:
		d.stats.TotalQueued++
		return true
	default:
		return false
	}
}

// worker processes products from the queue
func (d *DetailScraper) worker(id int) {
	defer d.wg.Done()

	log.Printf("[DetailScraper] Worker %d started", id)

	for {
		select {
		case <-d.stopCh:
			log.Printf("[DetailScraper] Worker %d stopping", id)
			return
		case product, ok := <-d.queue:
			if !ok {
				return
			}
			d.processWithRetry(product, id)
		}
	}
}

// processWithRetry processes a product with retry logic
func (d *DetailScraper) processWithRetry(product *model.Product, workerID int) {
	var lastErr error
	var updatedProduct *model.Product

	for attempt := 0; attempt <= d.retryMax; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s
			backoff := d.retryDelay * time.Duration(1<<uint(attempt-1))
			log.Printf("[DetailScraper] Worker %d: Retry %d/%d for %s after %v",
				workerID, attempt, d.retryMax, product.ID, backoff)
			time.Sleep(backoff)
			d.stats.TotalRetries++
		}

		// Fetch details
		updatedProduct = d.scraper.ScrapeProductDetails(product)

		// Save if we got a description
		if updatedProduct.Description != "" {
			d.store.UpsertProduct(updatedProduct)
			d.store.Save()
			d.stats.TotalSuccess++
			log.Printf("[DetailScraper] Worker %d: ✓ %s - %d chars",
				workerID, product.ID, len(updatedProduct.Description))
			d.stats.TotalProcessed++
			return
		}

		lastErr = fmt.Errorf("no description extracted")
	}

	// All retries exhausted
	d.stats.TotalFailed++
	d.stats.TotalProcessed++
	log.Printf("[DetailScraper] Worker %d: ✗ %s - failed after %d retries: %v",
		workerID, product.ID, d.retryMax, lastErr)
}

// statsReporter periodically logs statistics
func (d *DetailScraper) statsReporter() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.mu.Lock()
			stats := d.stats
			queueLen := len(d.queue)
			d.mu.Unlock()

			log.Printf("[DetailScraper] Stats - Queue: %d, Processed: %d, Success: %d, Failed: %d, Retries: %d",
				queueLen, stats.TotalProcessed, stats.TotalSuccess, stats.TotalFailed, stats.TotalRetries)
		}
	}
}

// GetStats returns current statistics
func (d *DetailScraper) GetStats() DetailStats {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.stats
}

// IsRunning returns whether the detail scraper is running
func (d *DetailScraper) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isRunning
}

// GetQueueSize returns the current queue size (thread-safe)
func (d *DetailScraper) GetQueueSize() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.queue)
}

// ProcessExistingProducts processes products that don't have descriptions yet
func (d *DetailScraper) ProcessExistingProducts() {
	products := d.store.GetAllProducts()

	needDetails := make([]*model.Product, 0)
	for _, p := range products {
		if p.Description == "" && p.ProductURL != "" {
			needDetails = append(needDetails, p)
		}
	}

	if len(needDetails) > 0 {
		log.Printf("[DetailScraper] Found %d existing products needing details", len(needDetails))
		queued := d.Enqueue(needDetails)
		log.Printf("[DetailScraper] Enqueued %d/%d existing products", queued, len(needDetails))
	}
}

// ScrapeWithAsyncDetails scrapes products and queues detail fetching asynchronously
func (d *DetailScraper) ScrapeWithAsyncDetails(ctx context.Context) error {
	log.Println("[DetailScraper] Starting scrape with async details...")

	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Scrape all products
	products, err := d.scraper.ScrapeAll()
	if err != nil {
		return fmt.Errorf("scrape failed: %w", err)
	}

	log.Printf("[DetailScraper] Scraped %d products", len(products))

	// Check for cancellation after scraping
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Upsert all products first (without details)
	for _, product := range products {
		d.store.UpsertProduct(product)
	}

	// Save initial data
	d.store.Save()

	// Enqueue products for detail fetching
	queued := d.Enqueue(products)
	log.Printf("[DetailScraper] Enqueued %d/%d products for async details", queued, len(products))

	return nil
}
