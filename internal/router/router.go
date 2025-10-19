package router

import (
	"context"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"isl-api-server/config"
	"isl-api-server/internal/models"
	"isl-api-server/internal/service"
	"isl-api-server/internal/util"
)

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config, logger *util.Logger) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create services
	healthService := service.NewHealthService()
	languageService := service.NewLanguageService()

	// Create translation service
	var translationService *service.TranslationService
	if cfg.GCPProjectID != "" {
		var err error
		translationService, err = service.NewTranslationService(cfg.GCPProjectID, cfg.LanguageDictionaryPath, logger)
		if err != nil {
			logger.Error("Failed to create translation service", "error", err)
		}
	} else {
		logger.Error("GCP Project ID not configured", "error", "GCP_PROJECT_ID environment variable not set")
	}

	// Create Gin engine
	r := gin.New()

	// Add middleware
	r.Use(gin.LoggerWithWriter(logger.GetLogFile()))
	r.Use(gin.RecoveryWithWriter(logger.GetLogFile()))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			healthData := healthService.GetHealthStatus()
			c.JSON(http.StatusOK, healthData)
		})

		v1.POST("/text-translate", func(c *gin.Context) {
			var req models.TranslationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "Invalid request",
					Message: err.Error(),
				})
				return
			}

			// Validate source language
			if !languageService.IsValidLanguage(req.SourceLanguage) {
				c.JSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "Invalid source language",
					Message: "Source language must be one of: en, hi, mr, gu",
				})
				return
			}

			// Get target languages
			targetLanguages := languageService.GetTargetLanguages(req.SourceLanguage)

			// Check if translation service is available
			if translationService == nil {
				c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
					Error:   "Translation service unavailable",
					Message: "GCP Translation service is not configured. Please set GCP_PROJECT_ID environment variable.",
				})
				return
			}

			// Translate to all target languages concurrently
			ctx := context.Background()
			translations, err := translationService.TranslateToMultipleLanguagesConcurrent(ctx, req.Text, req.SourceLanguage, targetLanguages)
			if err != nil {
				logger.Error("Translation failed", "error", err)
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "Translation failed",
					Message: "Unable to translate text. Please try again.",
				})
				return
			}

			// Return response
			response := models.TranslationResponse{
				OriginalText:   req.Text,
				SourceLanguage: req.SourceLanguage,
				Translations:   translations,
			}

			c.JSON(http.StatusOK, response)
		})
	}

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		response := gin.H{
			"message": "Welcome to ISL API Server",
			"version": "1.0.0",
			"endpoints": gin.H{
				"health":         "/api/v1/health",
				"text-translate": "/api/v1/text-translate",
			},
		}
		c.JSON(http.StatusOK, response)
	})

	r.NoRoute(func(c *gin.Context) {
		response := gin.H{
			"error":   "Not Found",
			"message": "The requested resource was not found",
			"path":    c.Request.URL.Path,
		}
		c.JSON(http.StatusNotFound, response)
	})

	return r
}
