package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"apple-price/internal/model"

	"github.com/gin-gonic/gin"
)

// StoreInterface defines the store interface needed by handlers
type StoreInterface interface {
	GetAllProducts() []*model.Product
	GetProduct(id string) (*model.Product, bool)
	GetProductsByCategory(category string) []*model.Product
	GetProductsByRegion(region string) []*model.Product
	GetPriceHistory(productID string) []model.PriceHistory
	GetCategories() []string
	AddSubscription(sub *model.Subscription) error
	RemoveSubscription(id string) error
	GetSubscriptionsByProduct(productID string) []*model.Subscription
	GetAllSubscriptions() []*model.Subscription
	GetStats() *model.Stats
	DeleteProductsByRegion(region string) (int, error)
	Save() error
	AddNewArrivalSubscription(sub *model.NewArrivalSubscription) error
	RemoveNewArrivalSubscription(id string) error
	GetAllNewArrivalSubscriptions() []*model.NewArrivalSubscription
	GetNewArrivalSubscription(id string) (*model.NewArrivalSubscription, bool)
}

// Handlers contains all API handlers
type Handlers struct {
	store      StoreInterface
	dispatcher PriceChangeNotifier
	scheduler  SchedulerInterface
}

// PriceChangeNotifier interface for handlers
type PriceChangeNotifier interface {
	NotifyPriceChange(product *model.Product, oldPrice, newPrice float64, subscriptions []*model.Subscription) error
}

// SchedulerInterface defines the scheduler interface for handlers
type SchedulerInterface interface {
	ScrapeNow() error
	GetScrapeStatus() any
}

// NewHandlers creates a new handlers instance
func NewHandlers(store StoreInterface, dispatcher PriceChangeNotifier, scheduler SchedulerInterface) *Handlers {
	return &Handlers{
		store:      store,
		dispatcher: dispatcher,
		scheduler:  scheduler,
	}
}

// HealthCheck returns the health status
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// GetProducts returns all products with optional filters
func (h *Handlers) GetProducts(c *gin.Context) {
	// Get filters
	category := c.Query("category")
	region := c.Query("region")
	sortBy := c.Query("sort") // price, discount, score, created
	order := c.Query("order") // asc, desc

	// Get products
	var products []*model.Product
	if category != "" && region != "" {
		// Filter by both
		allProducts := h.store.GetAllProducts()
		for _, p := range allProducts {
			if p.Category == category && p.Region == region {
				products = append(products, p)
			}
		}
	} else if category != "" {
		products = h.store.GetProductsByCategory(category)
	} else if region != "" {
		products = h.store.GetProductsByRegion(region)
	} else {
		products = h.store.GetAllProducts()
	}

	// Apply sorting
	products = sortProducts(products, sortBy, order)

	// Filter by stock status if requested
	if stockStatus := c.Query("stock_status"); stockStatus != "" {
		filtered := make([]*model.Product, 0)
		for _, p := range products {
			if p.StockStatus == stockStatus {
				filtered = append(filtered, p)
			}
		}
		products = filtered
	}

	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.JSON(http.StatusOK, gin.H{
		"count":    len(products),
		"products": products,
	})
}

// GetProduct returns a single product by ID
func (h *Handlers) GetProduct(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product ID is required"})
		return
	}

	product, ok := h.store.GetProduct(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// GetProductHistory returns price history for a product
func (h *Handlers) GetProductHistory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product ID is required"})
		return
	}

	_, ok := h.store.GetProduct(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	history := h.store.GetPriceHistory(id)

	// Parse limit parameter (capped at maxLimit)
	const maxLimit = 1000
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > maxLimit {
				limit = maxLimit
			} else {
				limit = l
			}
		}
	}

	// Apply limit
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id": id,
		"count":      len(history),
		"history":    history,
	})
}

// CreateSubscription creates a new subscription
func (h *Handlers) CreateSubscription(c *gin.Context) {
	var req struct {
		ProductID  string  `json:"product_id" binding:"required"`
		BarkKey    string  `json:"bark_key"`
		Email      string  `json:"email"`
		TargetPrice float64 `json:"target_price"` // Optional target price for alert
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate product exists
	_, ok := h.store.GetProduct(req.ProductID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	// Create subscription
	sub := &model.Subscription{
		ID:         generateID(),
		ProductID:  req.ProductID,
		BarkKey:    req.BarkKey,
		Email:      req.Email,
		TargetPrice: req.TargetPrice,
		CreatedAt:  time.Now(),
	}

	if err := h.store.AddSubscription(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	// Save to disk
	if err := h.store.Save(); err != nil {
		// Log error but don't fail the request
		// The subscription is in memory
	}

	c.JSON(http.StatusCreated, sub)
}

// DeleteSubscription deletes a subscription
func (h *Handlers) DeleteSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription ID is required"})
		return
	}

	if err := h.store.RemoveSubscription(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	// Save to disk
	if err := h.store.Save(); err != nil {
		// Log error but don't fail the request
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription deleted"})
}

// GetSubscriptions returns all subscriptions for a product
func (h *Handlers) GetSubscriptions(c *gin.Context) {
	productID := c.Query("product_id")

	if productID != "" {
		subs := h.store.GetSubscriptionsByProduct(productID)
		c.JSON(http.StatusOK, gin.H{
			"count":         len(subs),
			"subscriptions": subs,
		})
	} else {
		subs := h.store.GetAllSubscriptions()
		c.JSON(http.StatusOK, gin.H{
			"count":         len(subs),
			"subscriptions": subs,
		})
	}
}

// GetCategories returns all product categories
func (h *Handlers) GetCategories(c *gin.Context) {
	categories := h.store.GetCategories()

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
	})
}

// GetStats returns system statistics
func (h *Handlers) GetStats(c *gin.Context) {
	stats := h.store.GetStats()

	c.JSON(http.StatusOK, stats)
}

// TriggerScrape triggers a manual scrape
func (h *Handlers) TriggerScrape(c *gin.Context) {
	// Trigger scrape through scheduler
	if h.scheduler != nil {
		go func() {
			_ = h.scheduler.ScrapeNow()
		}()
		c.JSON(http.StatusAccepted, gin.H{
			"message": "scrape triggered",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "scheduler not available",
		})
	}
}

// GetDetailStatus returns the detail scraper status
func (h *Handlers) GetDetailStatus(c *gin.Context) {
	if h.scheduler != nil {
		status := h.scheduler.GetScrapeStatus()
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "scheduler not available",
		})
	}
}

// DeleteProductsByRegion deletes all products from a specific region
func (h *Handlers) DeleteProductsByRegion(c *gin.Context) {
	region := c.Param("region")
	if region == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region is required"})
		return
	}

	count, err := h.store.DeleteProductsByRegion(region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete products"})
		return
	}

	if err := h.store.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Deleted %d products from region %s", count, region),
		"count":   count,
	})
}

// sortProducts sorts products based on the given criteria
func sortProducts(products []*model.Product, sortBy, order string) []*model.Product {
	if len(products) <= 1 {
		return products
	}

	// Make a copy to avoid mutating the original
	sorted := make([]*model.Product, len(products))
	copy(sorted, products)

	// Sort based on the sortBy parameter
	switch sortBy {
	case "price":
		sortByPrice(sorted, order == "desc")
	case "discount":
		sortByDiscount(sorted, order == "desc")
	case "score":
		sortByScore(sorted, order == "desc")
	case "created":
		sortByCreated(sorted, order == "desc")
	default:
		// Default: sort by score descending
		sortByScore(sorted, true)
	}

	return sorted
}

// sortByPrice sorts products by price
func sortByPrice(products []*model.Product, desc bool) {
	if desc {
		sort.Slice(products, func(i, j int) bool {
			return products[i].Price > products[j].Price
		})
	} else {
		sort.Slice(products, func(i, j int) bool {
			return products[i].Price < products[j].Price
		})
	}
}

// sortByDiscount sorts products by discount
func sortByDiscount(products []*model.Product, desc bool) {
	if desc {
		sort.Slice(products, func(i, j int) bool {
			return products[i].Discount > products[j].Discount
		})
	} else {
		sort.Slice(products, func(i, j int) bool {
			return products[i].Discount < products[j].Discount
		})
	}
}

// sortByScore sorts products by value score
func sortByScore(products []*model.Product, desc bool) {
	if desc {
		sort.Slice(products, func(i, j int) bool {
			return products[i].ValueScore > products[j].ValueScore
		})
	} else {
		sort.Slice(products, func(i, j int) bool {
			return products[i].ValueScore < products[j].ValueScore
		})
	}
}

// sortByCreated sorts products by creation time
func sortByCreated(products []*model.Product, desc bool) {
	if desc {
		sort.Slice(products, func(i, j int) bool {
			return products[i].CreatedAt.After(products[j].CreatedAt)
		})
	} else {
		sort.Slice(products, func(i, j int) bool {
			return products[i].CreatedAt.Before(products[j].CreatedAt)
		})
	}
}

// generateID generates a unique ID
func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// CreateNewArrivalSubscription creates a new arrival subscription
func (h *Handlers) CreateNewArrivalSubscription(c *gin.Context) {
	var req model.NewArrivalSubscription
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.BarkKey == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either bark_key or email is required"})
		return
	}

	// Generate ID and set defaults
	req.ID = generateID()
	req.CreatedAt = time.Now()
	if !req.Enabled {
		req.Enabled = true
	}

	if err := h.store.AddNewArrivalSubscription(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save subscription"})
		return
	}

	if err := h.store.Save(); err != nil {
		// Log error but don't fail
	}

	c.JSON(http.StatusCreated, req)
}

// DeleteNewArrivalSubscription deletes a new arrival subscription
func (h *Handlers) DeleteNewArrivalSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if err := h.store.RemoveNewArrivalSubscription(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	if err := h.store.Save(); err != nil {
		// Log error but don't fail
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription deleted"})
}

// GetNewArrivalSubscriptions returns all new arrival subscriptions
func (h *Handlers) GetNewArrivalSubscriptions(c *gin.Context) {
	subs := h.store.GetAllNewArrivalSubscriptions()
	c.JSON(http.StatusOK, gin.H{
		"count":         len(subs),
		"subscriptions": subs,
	})
}

// GetNewArrivalSubscription returns a single new arrival subscription
func (h *Handlers) GetNewArrivalSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	sub, found := h.store.GetNewArrivalSubscription(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}
