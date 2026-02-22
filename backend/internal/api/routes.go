package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(r *gin.Engine, store StoreInterface, dispatcher PriceChangeNotifier, scheduler SchedulerInterface) {
	handlers := NewHandlers(store, dispatcher, scheduler)

	// API v1 routes
	v1 := r.Group("/api")
	{
		// Health check (handle both GET and HEAD)
		v1.GET("/health", handlers.HealthCheck)
		v1.HEAD("/health", handlers.HealthCheck)

		// Products
		v1.GET("/products", handlers.GetProducts)
		v1.GET("/products/:id", handlers.GetProduct)
		v1.GET("/products/:id/history", handlers.GetProductHistory)

		// Subscriptions
		v1.POST("/subscriptions", handlers.CreateSubscription)
		v1.DELETE("/subscriptions/:id", handlers.DeleteSubscription)
		v1.GET("/subscriptions", handlers.GetSubscriptions)

		// New Arrival Subscriptions
		v1.POST("/new-arrival-subscriptions", handlers.CreateNewArrivalSubscription)
		v1.DELETE("/new-arrival-subscriptions/:id", handlers.DeleteNewArrivalSubscription)
		v1.GET("/new-arrival-subscriptions", handlers.GetNewArrivalSubscriptions)
		v1.GET("/new-arrival-subscriptions/:id", handlers.GetNewArrivalSubscription)
		v1.PUT("/new-arrival-subscriptions/:id", handlers.UpdateNewArrivalSubscription)
		v1.PATCH("/new-arrival-subscriptions/:id/pause", handlers.PauseSubscription)
		v1.PATCH("/new-arrival-subscriptions/:id/resume", handlers.ResumeSubscription)

		// Notification History
		v1.GET("/notification-history", handlers.GetNotificationHistory)
		v1.POST("/notification-history/:id/read", handlers.MarkNotificationAsRead)
		v1.GET("/notification-history/unread-count", handlers.GetUnreadNotificationCount)

		// Categories
		v1.GET("/categories", handlers.GetCategories)

		// Filter Options
		v1.GET("/filter-options", handlers.GetFilterOptions)

		// Stats
		v1.GET("/stats", handlers.GetStats)

		// Recommendations (断层领先: 智能推荐)
		v1.POST("/recommendations", handlers.HandleRecommendation)

		// Detail scraper status
		v1.GET("/admin/detail-status", handlers.GetDetailStatus)

		// Admin operations (WARNING: No authentication - add auth middleware before production)
		v1.POST("/admin/scrape", handlers.TriggerScrape)
		v1.DELETE("/admin/products/region/:region", handlers.DeleteProductsByRegion)
	}

	// Serve frontend static files in production
	// r.Static("/", "./frontend/dist")
}
