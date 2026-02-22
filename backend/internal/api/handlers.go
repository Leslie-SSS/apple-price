package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
	GetNewArrivalSubscriptionsByBarkKey(barkKey string) []*model.NewArrivalSubscription
	GetNewArrivalSubscription(id string) (*model.NewArrivalSubscription, bool)

	// Notification history operations
	AddNotificationHistory(history *model.NotificationHistory) error
	GetNotificationHistory(subscriptionID string, barkKey string, limit, offset int) ([]*model.NotificationHistory, int)
	MarkNotificationAsRead(id string) error
	GetUnreadNotificationCount() int

	// Subscription management operations
	UpdateNewArrivalSubscription(sub *model.NewArrivalSubscription) error
	PauseSubscription(id string) error
	ResumeSubscription(id string) error
	IncrementNotificationCount(id string) error
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
		ProductID   string  `json:"product_id" binding:"required"`
		BarkKey     string  `json:"bark_key" binding:"required"`
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
		ID:          generateID(),
		ProductID:   req.ProductID,
		BarkKey:     req.BarkKey,
		TargetPrice: req.TargetPrice,
		CreatedAt:   time.Now(),
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

// GetFilterOptions returns dynamic filter options based on current products
func (h *Handlers) GetFilterOptions(c *gin.Context) {
	category := c.Query("category")

	// Get products based on category filter
	var products []*model.Product
	if category != "" && category != "全部" {
		products = h.store.GetProductsByCategory(category)
	} else {
		products = h.store.GetAllProducts()
	}

	options := extractFilterOptions(products)
	c.JSON(http.StatusOK, options)
}

// FilterOptions represents available filter options
type FilterOptions struct {
	Chips       []string `json:"chips"`
	Storages    []string `json:"storages"`
	Memories    []string `json:"memories"`
	ScreenSizes []string `json:"screen_sizes"`
	Colors      []string `json:"colors"`
	Models      []string `json:"models"`
}

func extractFilterOptions(products []*model.Product) FilterOptions {
	chips := make(map[string]bool)
	storages := make(map[string]bool)
	memories := make(map[string]bool)
	screenSizes := make(map[string]bool)
	colors := make(map[string]bool)
	models := make(map[string]bool)

	for _, p := range products {
		// Parse specs_detail JSON
		if p.SpecsDetail != "" {
			var specs model.ParsedSpecs
			if err := json.Unmarshal([]byte(p.SpecsDetail), &specs); err == nil {
				if specs.Chip != "" {
					chips[specs.Chip] = true
				}
				if specs.Storage != "" {
					storages[specs.Storage] = true
				}
				if specs.Memory != "" {
					memories[specs.Memory] = true
				}
				if specs.ScreenSize != "" {
					screenSizes[specs.ScreenSize] = true
				}
				if specs.Color != "" {
					colors[specs.Color] = true
				}
			}
		}

		// Extract model from name
		model := extractModelFromName(p.Name, p.Category)
		if model != "" {
			models[model] = true
		}
	}

	return FilterOptions{
		Chips:       sortByChipVersion(mapKeys(chips)),
		Storages:    sortByCapacity(mapKeys(storages)),
		Memories:    sortByCapacity(mapKeys(memories)),
		ScreenSizes: sortByScreenSize(mapKeys(screenSizes)),
		Colors:      sortColors(mapKeys(colors)),
		Models:      sortModels(mapKeys(models)),
	}
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func extractModelFromName(name, category string) string {
	nameLower := strings.ToLower(name)
	switch category {
	case "Mac":
		switch {
		case strings.Contains(nameLower, "macbook air"):
			return "MacBook Air"
		case strings.Contains(nameLower, "macbook pro"):
			return "MacBook Pro"
		case strings.Contains(nameLower, "mac mini"):
			return "Mac mini"
		case strings.Contains(nameLower, "mac studio"):
			return "Mac Studio"
		case strings.Contains(nameLower, "imac"):
			return "iMac"
		case strings.Contains(nameLower, "mac pro"):
			return "Mac Pro"
		}
	case "iPad":
		switch {
		case strings.Contains(nameLower, "ipad pro"):
			return "iPad Pro"
		case strings.Contains(nameLower, "ipad air"):
			return "iPad Air"
		case strings.Contains(nameLower, "ipad mini"):
			return "iPad mini"
		case strings.Contains(nameLower, "ipad"):
			return "iPad"
		}
	case "Watch":
		switch {
		case strings.Contains(nameLower, "ultra"):
			return "Apple Watch Ultra"
		case strings.Contains(nameLower, "series"):
			return "Apple Watch Series"
		case strings.Contains(nameLower, "se"):
			return "Apple Watch SE"
		}
	}
	return ""
}

func sortByChipVersion(chips []string) []string {
	sort.Slice(chips, func(i, j int) bool {
		// Sort by chip generation and tier (M4 > M3 > M2 > M1, Pro > base)
		getChipOrder := func(chip string) (gen, tier int) {
			chipLower := strings.ToLower(chip)
			// Generation
			switch {
			case strings.Contains(chipLower, "m4"):
				gen = 4
			case strings.Contains(chipLower, "m3"):
				gen = 3
			case strings.Contains(chipLower, "m2"):
				gen = 2
			case strings.Contains(chipLower, "m1"):
				gen = 1
			}
			// Tier
			switch {
			case strings.Contains(chipLower, "ultra"):
				tier = 4
			case strings.Contains(chipLower, "max"):
				tier = 3
			case strings.Contains(chipLower, "pro"):
				tier = 2
			default:
				tier = 1
			}
			return
		}
		genI, tierI := getChipOrder(chips[i])
		genJ, tierJ := getChipOrder(chips[j])
		if genI != genJ {
			return genI > genJ
		}
		return tierI > tierJ
	})
	return chips
}

func sortByCapacity(caps []string) []string {
	parseCapacity := func(s string) int {
		s = strings.ToLower(s)
		var value int
		if strings.Contains(s, "tb") {
			fmt.Sscanf(s, "%d", &value)
			return value * 1024
		}
		fmt.Sscanf(s, "%d", &value)
		return value
	}
	sort.Slice(caps, func(i, j int) bool {
		return parseCapacity(caps[i]) < parseCapacity(caps[j])
	})
	return caps
}

func sortByScreenSize(sizes []string) []string {
	parseSize := func(s string) float64 {
		var size float64
		fmt.Sscanf(s, "%f", &size)
		return size
	}
	sort.Slice(sizes, func(i, j int) bool {
		return parseSize(sizes[i]) < parseSize(sizes[j])
	})
	return sizes
}

func sortColors(colors []string) []string {
	// Common color order
	colorOrder := map[string]int{
		"深空黑": 1, "深空灰": 2, "银色": 3, "金色": 4, "星光色": 5,
		"午夜色": 6, "蓝色": 7, "紫色": 8, "绿色": 9, "粉色": 10,
		"红色": 11, "黑色": 12, "白色": 13, "玫瑰金": 14,
	}
	sort.Slice(colors, func(i, j int) bool {
		orderI, okI := colorOrder[colors[i]]
		orderJ, okJ := colorOrder[colors[j]]
		if okI && okJ {
			return orderI < orderJ
		}
		if okI {
			return true
		}
		return colors[i] < colors[j]
	})
	return colors
}

func sortModels(models []string) []string {
	sort.Strings(models)
	return models
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

	// Bark Key is required for each subscription
	if req.BarkKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bark Key 是必填项"})
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

	// Return subscription with masked Bark Key
	response := req
	response.BarkKey = maskBarkKey(response.BarkKey)
	c.JSON(http.StatusCreated, response)
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

// GetNewArrivalSubscriptions returns new arrival subscriptions for a specific Bark Key
func (h *Handlers) GetNewArrivalSubscriptions(c *gin.Context) {
	barkKey := c.Query("bark_key")

	if barkKey == "" {
		// No Bark Key provided, return empty list
		c.JSON(http.StatusOK, gin.H{
			"count":         0,
			"subscriptions": []*model.NewArrivalSubscription{},
		})
		return
	}

	subs := h.store.GetNewArrivalSubscriptionsByBarkKey(barkKey)

	// Mask Bark Key in response for privacy
	for _, sub := range subs {
		sub.BarkKey = maskBarkKey(sub.BarkKey)
	}

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

// GetNotificationHistory returns notification history with pagination
func (h *Handlers) GetNotificationHistory(c *gin.Context) {
	// Get query parameters
	subscriptionID := c.Query("subscription_id")
	barkKey := c.Query("bark_key") // Filter by Bark Key for user isolation
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	// Parse limit (max 200)
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Parse offset
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// If no bark_key provided, return empty (user isolation)
	if barkKey == "" {
		c.JSON(http.StatusOK, gin.H{
			"data":   []*model.NotificationHistory{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	history, total := h.store.GetNotificationHistory(subscriptionID, barkKey, limit, offset)

	c.JSON(http.StatusOK, gin.H{
		"data":  history,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// MarkNotificationAsRead marks a notification as read
func (h *Handlers) MarkNotificationAsRead(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if err := h.store.MarkNotificationAsRead(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	if err := h.store.Save(); err != nil {
		// Log error but don't fail
	}

	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

// GetUnreadNotificationCount returns the count of unread notifications
func (h *Handlers) GetUnreadNotificationCount(c *gin.Context) {
	count := h.store.GetUnreadNotificationCount()
	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// UpdateNewArrivalSubscription updates an existing subscription
func (h *Handlers) UpdateNewArrivalSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	// Check if subscription exists
	existing, found := h.store.GetNewArrivalSubscription(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	var req model.NewArrivalSubscription
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Preserve ID, Bark Key and timestamps
	req.ID = id
	req.BarkKey = existing.BarkKey // Preserve original Bark Key
	req.UpdatedAt = time.Now()

	if err := h.store.UpdateNewArrivalSubscription(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription"})
		return
	}

	if err := h.store.Save(); err != nil {
		// Log error but don't fail
	}

	// Return with masked Bark Key
	response := req
	response.BarkKey = maskBarkKey(response.BarkKey)
	c.JSON(http.StatusOK, response)
}

// PauseSubscription pauses a subscription
func (h *Handlers) PauseSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	// Check if subscription exists
	_, found := h.store.GetNewArrivalSubscription(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	if err := h.store.PauseSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to pause subscription"})
		return
	}

	if err := h.store.Save(); err != nil {
		// Log error but don't fail
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription paused"})
}

// ResumeSubscription resumes a paused subscription
func (h *Handlers) ResumeSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	// Check if subscription exists
	_, found := h.store.GetNewArrivalSubscription(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	if err := h.store.ResumeSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resume subscription"})
		return
	}

	if err := h.store.Save(); err != nil {
		// Log error but don't fail
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription resumed"})
}

// maskBarkKey masks a Bark Key for display (shows first 4 and last 4 chars)
func maskBarkKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
