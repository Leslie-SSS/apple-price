package scraper

import (
	"log"
	"time"

	"apple-price/internal/model"
)

// Scheduler manages periodic scraping
type Scheduler struct {
	scraper       Scraper
	detailScraper *DetailScraper
	store         StoreInterface
	notifier      PriceChangeNotifier
	interval      time.Duration
	stopCh        chan struct{}
	isRunning     bool
}

// StoreInterface defines the store interface needed by scheduler
// This allows both old JSON store and new SQLite store to work
type StoreInterface interface {
	UpsertProduct(product *model.Product) (priceChanged bool, oldPrice float64)
	GetProduct(id string) (*model.Product, bool)
	GetSubscriptionsByProduct(productID string) []*model.Subscription
	GetAllNewArrivalSubscriptions() []*model.NewArrivalSubscription
	UpdateNotifiedProductIDs(subscriptionID, productID string) error
	UpdateLastScrapeTime(t time.Time)
	GetLastScrapeTime() time.Time
	Save() error
	GetAllProducts() []*model.Product
	GetScraperStatus() *model.ScraperStatus
	UpdateScraperStatus(status *model.ScraperStatus) error
}

// PriceChangeNotifier interface for price change notifications
type PriceChangeNotifier interface {
	NotifyPriceChange(product *model.Product, oldPrice, newPrice float64, subscriptions []*model.Subscription) error
	NotifyNewArrival(product *model.Product, subscriptions []*model.NewArrivalSubscription) error
}

// NewScheduler creates a new scheduler
func NewScheduler(
	scraper Scraper,
	store StoreInterface,
	notifier PriceChangeNotifier,
	interval time.Duration,
) *Scheduler {
	return &Scheduler{
		scraper:  scraper,
		store:    store,
		notifier: notifier,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// SetDetailScraper sets the detail scraper for async detail fetching
func (s *Scheduler) SetDetailScraper(ds *DetailScraper) {
	s.detailScraper = ds
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	if s.isRunning {
		log.Println("Scheduler already running")
		return
	}

	s.isRunning = true
	log.Printf("Scheduler started with interval: %v", s.interval)

	// Start detail scraper if available
	if s.detailScraper != nil {
		s.detailScraper.Start()
		// Process existing products that need details
		go func() {
			time.Sleep(5 * time.Second) // Wait for initial scrape
			s.detailScraper.ProcessExistingProducts()
		}()
	}

	// Run immediately on start
	s.runScrape()

	// Start ticker
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runScrape()
			case <-s.stopCh:
				log.Println("Scheduler stopped")
				s.isRunning = false

				// Stop detail scraper
				if s.detailScraper != nil {
					s.detailScraper.Stop()
				}

				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.isRunning {
		close(s.stopCh)

		// Stop detail scraper
		if s.detailScraper != nil {
			s.detailScraper.Stop()
		}
	}
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	return s.isRunning
}

// runScrape executes a single scrape cycle
func (s *Scheduler) runScrape() {
	startTime := time.Now()
	log.Println("Starting scrape cycle...")

	// Record running status
	s.store.UpdateScraperStatus(&model.ScraperStatus{
		LastScrapeTime:   startTime,
		LastScrapeStatus: "running",
	})

	products, err := s.scraper.ScrapeAll()
	if err != nil {
		log.Printf("Scrape error: %v", err)
		// Record failed status
		s.store.UpdateScraperStatus(&model.ScraperStatus{
			LastScrapeTime:   startTime,
			LastScrapeStatus: "failed",
			LastScrapeError:  err.Error(),
		})
		return
	}

	log.Printf("Scraped %d products", len(products))

	// Upsert all products and track price changes
	priceChangeCount := 0
	newProductCount := 0

	for _, product := range products {
		priceChanged, oldPrice := s.store.UpsertProduct(product)

		// Check if this is a new product (oldPrice == 0 and no price change)
		isNewProduct := !priceChanged && oldPrice == 0

		if priceChanged && s.notifier != nil {
			priceChangeCount++
			log.Printf("Price changed for %s: %.2f -> %.2f", product.Name, oldPrice, product.Price)

			// Get subscriptions for this product
			subscriptions := s.store.GetSubscriptionsByProduct(product.ID)

			// Notify subscribers
			if err := s.notifier.NotifyPriceChange(product, oldPrice, product.Price, subscriptions); err != nil {
				log.Printf("Failed to notify price change: %v", err)
			}
		}

		// Notify new arrival subscribers for new products
		if isNewProduct && s.notifier != nil {
			newProductCount++
			log.Printf("New product detected: %s (%s)", product.Name, product.Category)

			// Get all new arrival subscriptions
			arrivalSubscriptions := s.store.GetAllNewArrivalSubscriptions()

			// Notify matching subscribers
			if err := s.notifier.NotifyNewArrival(product, arrivalSubscriptions); err != nil {
				log.Printf("Failed to notify new arrival: %v", err)
			}

			// Update notified_product_ids for subscriptions that matched
			// This is done inside NotifyNewArrival via the dispatcher
		}
	}

	// Update last scrape time
	s.store.UpdateLastScrapeTime(time.Now())

	// Save data to disk
	if err := s.store.Save(); err != nil {
		log.Printf("Failed to save data: %v", err)
	}

	// Enqueue products for async detail fetching
	if s.detailScraper != nil {
		queued := s.detailScraper.Enqueue(products)
		if queued > 0 {
			log.Printf("[Scheduler] Enqueued %d products for async detail fetching", queued)
		}
	}

	duration := time.Since(startTime)
	log.Printf("Scrape cycle completed in %v. Products: %d, Price changes: %d, New products: %d",
		duration, len(products), priceChangeCount, newProductCount)

	// Record success status
	s.store.UpdateScraperStatus(&model.ScraperStatus{
		LastScrapeTime:   time.Now(),
		LastScrapeStatus: "success",
		ProductsScraped:  len(products),
		Duration:         duration.Milliseconds(),
	})
}

// ScrapeNow triggers an immediate scrape
func (s *Scheduler) ScrapeNow() error {
	s.runScrape()
	return nil
}

// GetScrapeStatus returns the current status of the scheduler
func (s *Scheduler) GetScrapeStatus() any {
	status := &ScrapeStatus{
		IsRunning:      s.isRunning,
		Interval:       s.interval,
		LastScrapeTime: s.store.GetLastScrapeTime(),
	}

	if s.detailScraper != nil {
		stats := s.detailScraper.GetStats()
		status.DetailStats = &stats
		status.DetailQueueSize = s.detailScraper.GetQueueSize()
	}

	return status
}

// ScrapeStatus represents the scheduler status
type ScrapeStatus struct {
	IsRunning       bool          `json:"is_running"`
	Interval        time.Duration `json:"interval"`
	LastScrapeTime  time.Time     `json:"last_scrape_time"`
	DetailStats     *DetailStats  `json:"detail_stats,omitempty"`
	DetailQueueSize int          `json:"detail_queue_size,omitempty"`
}
