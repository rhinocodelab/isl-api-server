package router

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"isl-api-server/config"
	"isl-api-server/internal/models"
	"isl-api-server/internal/service"
	"isl-api-server/internal/util"
)

// generateUniqueFileName generates a unique filename to prevent collisions
func generateUniqueFileName(originalName string) string {
	timestamp := time.Now().UnixNano()
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)
	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}

// applyTrainNameCorrection applies train name correction to text
func applyTrainNameCorrection(text, languageCode string) string {
	// Train names mapping
	trainNames := map[string]map[string]string{
		"Tata": {
			"en-IN": "Tejas",
			"hi-IN": "तेजस",
			"mr-IN": "तेजस",
			"gu-IN": "તેજસ",
		},
		"टाटा": {
			"en-IN": "Tejas",
			"hi-IN": "तेजस",
			"mr-IN": "तेजस",
			"gu-IN": "તેજસ",
		},
		"ટાટા": {
			"en-IN": "Tejas",
			"hi-IN": "तेजस",
			"mr-IN": "तेजस",
			"gu-IN": "તેજસ",
		},
		"टेटस": {
			"en-IN": "Tejas",
			"hi-IN": "तेजस",
			"mr-IN": "तेजस",
			"gu-IN": "તેજસ",
		},
	}

	// Apply corrections
	for trainName, translations := range trainNames {
		// Check if the train name exists in the text
		if strings.Contains(strings.ToLower(text), strings.ToLower(trainName)) {
			// Get the correct translation for the target language
			if correctName, exists := translations[languageCode]; exists {
				// Replace the train name with the correct translation
				text = strings.Replace(text, trainName, correctName, -1)
			}
		}

		// Also check for translated train names in the text
		for lang, translatedName := range translations {
			if lang != languageCode && strings.Contains(strings.ToLower(text), strings.ToLower(translatedName)) {
				// Get the correct translation for the target language
				if correctName, exists := translations[languageCode]; exists {
					// Replace the translated train name with the correct translation
					text = strings.Replace(text, translatedName, correctName, -1)
				}
			}
		}
	}

	return text
}

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

	// Create audio language detection service
	var audioDetectionService *service.AudioLanguageDetectionService
	if cfg.GeminiAPIKey != "" {
		var err error
		audioDetectionService, err = service.NewAudioLanguageDetectionService(cfg.GeminiAPIKey, logger)
		if err != nil {
			logger.Error("Failed to create audio language detection service", "error", err)
		}
	} else {
		logger.Error("Gemini API Key not configured", "error", "GEMINI_API_KEY environment variable not set")
	}

	// Create speech-to-text service
	var speechToTextService *service.SpeechToTextService
	if cfg.GCPProjectID != "" {
		var err error
		speechToTextService, err = service.NewSpeechToTextService(cfg.GCPProjectID, cfg.LanguageDictionaryPath, logger)
		if err != nil {
			logger.Error("Failed to create speech-to-text service", "error", err)
		}
	} else {
		logger.Error("GCP Project ID not configured for speech-to-text", "error", "GCP_PROJECT_ID environment variable not set")
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

		v1.POST("/audio-language-detect", func(c *gin.Context) {
			// Check if audio detection service is available
			if audioDetectionService == nil {
				c.JSON(http.StatusServiceUnavailable, models.AudioLanguageDetectionError{
					Error:   "Audio language detection service unavailable",
					Message: "Gemini API service is not configured. Please set GEMINI_API_KEY environment variable.",
					Code:    "SERVICE_UNAVAILABLE",
				})
				return
			}

			// Get the uploaded file
			file, err := c.FormFile("audio_file")
			if err != nil {
				c.JSON(http.StatusBadRequest, models.AudioLanguageDetectionError{
					Error:   "Invalid file upload",
					Message: "Please upload an audio file with the field name 'audio_file'",
					Code:    "INVALID_FILE",
				})
				return
			}

			// Validate file size (max 10MB)
			if file.Size > 10*1024*1024 {
				c.JSON(http.StatusBadRequest, models.AudioLanguageDetectionError{
					Error:   "File too large",
					Message: "Audio file size must be less than 10MB",
					Code:    "FILE_TOO_LARGE",
				})
				return
			}

			// Validate file format
			contentType := file.Header.Get("Content-Type")
			supportedFormats := audioDetectionService.GetSupportedAudioFormats()
			isValidFormat := false

			// Check MIME type first
			for _, format := range supportedFormats {
				if contentType == format {
					isValidFormat = true
					break
				}
			}

			// If MIME type validation fails, check file extension as fallback
			if !isValidFormat {
				fileExt := strings.ToLower(filepath.Ext(file.Filename))
				supportedExtensions := []string{".wav", ".mp3", ".aiff", ".aac", ".ogg", ".flac"}
				for _, ext := range supportedExtensions {
					if fileExt == ext {
						isValidFormat = true
						break
					}
				}
			}

			if !isValidFormat {
				c.JSON(http.StatusBadRequest, models.AudioLanguageDetectionError{
					Error:   "Unsupported audio format",
					Message: fmt.Sprintf("Supported formats: %v", supportedFormats),
					Code:    "UNSUPPORTED_FORMAT",
				})
				return
			}

			// Save uploaded file temporarily
			tempDir := "./temp"
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				logger.Error("Failed to create temp directory", "error", err)
				c.JSON(http.StatusInternalServerError, models.AudioLanguageDetectionError{
					Error:   "Internal server error",
					Message: "Failed to create temporary directory",
					Code:    "TEMP_DIR_ERROR",
				})
				return
			}

			// Generate unique filename to prevent collisions
			uniqueFileName := generateUniqueFileName(file.Filename)
			tempFilePath := fmt.Sprintf("%s/%s", tempDir, uniqueFileName)
			if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
				logger.Error("Failed to save uploaded file", "error", err)
				c.JSON(http.StatusInternalServerError, models.AudioLanguageDetectionError{
					Error:   "Internal server error",
					Message: "Failed to save uploaded file",
					Code:    "FILE_SAVE_ERROR",
				})
				return
			}

			// Clean up temporary file after processing
			defer func() {
				if err := os.Remove(tempFilePath); err != nil {
					logger.Error("Failed to remove temporary file", "error", err, "file", tempFilePath)
				}
			}()

			// Validate the saved file
			if err := audioDetectionService.ValidateAudioFile(tempFilePath); err != nil {
				c.JSON(http.StatusBadRequest, models.AudioLanguageDetectionError{
					Error:   "Invalid audio file",
					Message: err.Error(),
					Code:    "INVALID_AUDIO_FILE",
				})
				return
			}

			// Detect language using Gemini
			ctx := context.Background()
			detectedLanguage, err := audioDetectionService.DetectLanguage(ctx, tempFilePath)
			if err != nil {
				logger.Error("Audio language detection failed", "error", err)
				c.JSON(http.StatusInternalServerError, models.AudioLanguageDetectionError{
					Error:   "Language detection failed",
					Message: "Unable to detect language from audio file. Please try again.",
					Code:    "DETECTION_FAILED",
				})
				return
			}

			// Get language code
			languageCode := audioDetectionService.GetLanguageCode(detectedLanguage)

			// Return response
			response := models.AudioLanguageDetectionResponse{
				DetectedLanguage: detectedLanguage,
				LanguageCode:     languageCode,
				AudioFormat:      contentType,
				FileSize:         file.Size,
			}

			c.JSON(http.StatusOK, response)
		})

		v1.POST("/speech-to-text", func(c *gin.Context) {
			// Check if speech-to-text service is available
			if speechToTextService == nil {
				c.JSON(http.StatusServiceUnavailable, models.SpeechToTextError{
					Error:   "Speech-to-text service unavailable",
					Message: "GCP Speech-to-Text service is not configured. Please set GCP_PROJECT_ID environment variable.",
					Code:    "SERVICE_UNAVAILABLE",
				})
				return
			}

			// Check if translation service is available
			if translationService == nil {
				c.JSON(http.StatusServiceUnavailable, models.SpeechToTextError{
					Error:   "Translation service unavailable",
					Message: "GCP Translation service is not configured. Please set GCP_PROJECT_ID environment variable.",
					Code:    "TRANSLATION_SERVICE_UNAVAILABLE",
				})
				return
			}

			// Get the uploaded file
			file, err := c.FormFile("audio_file")
			if err != nil {
				c.JSON(http.StatusBadRequest, models.SpeechToTextError{
					Error:   "Invalid file upload",
					Message: "Please upload an audio file with the field name 'audio_file'",
					Code:    "INVALID_FILE",
				})
				return
			}

			// Get language code from form
			languageCode := c.PostForm("language_code")
			if languageCode == "" {
				c.JSON(http.StatusBadRequest, models.SpeechToTextError{
					Error:   "Missing language code",
					Message: "Please provide language_code in the form data",
					Code:    "MISSING_LANGUAGE_CODE",
				})
				return
			}

			// Validate language code
			if !speechToTextService.ValidateLanguageCode(languageCode) {
				c.JSON(http.StatusBadRequest, models.SpeechToTextError{
					Error:   "Invalid language code",
					Message: "Language code must be one of: en-IN, hi-IN, mr-IN, gu-IN",
					Code:    "INVALID_LANGUAGE_CODE",
				})
				return
			}

			// Validate file size (max 10MB)
			if file.Size > 10*1024*1024 {
				c.JSON(http.StatusBadRequest, models.SpeechToTextError{
					Error:   "File too large",
					Message: "Audio file size must be less than 10MB",
					Code:    "FILE_TOO_LARGE",
				})
				return
			}

			// Validate file format
			contentType := file.Header.Get("Content-Type")
			supportedFormats := []string{"audio/wav", "audio/mp3", "audio/aiff", "audio/aac", "audio/ogg", "audio/flac"}
			isValidFormat := false

			// Check MIME type first
			for _, format := range supportedFormats {
				if contentType == format {
					isValidFormat = true
					break
				}
			}

			// If MIME type validation fails, check file extension as fallback
			if !isValidFormat {
				fileExt := strings.ToLower(filepath.Ext(file.Filename))
				supportedExtensions := []string{".wav", ".mp3", ".aiff", ".aac", ".ogg", ".flac"}
				for _, ext := range supportedExtensions {
					if fileExt == ext {
						isValidFormat = true
						break
					}
				}
			}

			if !isValidFormat {
				c.JSON(http.StatusBadRequest, models.SpeechToTextError{
					Error:   "Unsupported audio format",
					Message: fmt.Sprintf("Supported formats: %v", supportedFormats),
					Code:    "UNSUPPORTED_FORMAT",
				})
				return
			}

			// Save uploaded file temporarily
			tempDir := "./temp"
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				logger.Error("Failed to create temp directory", "error", err)
				c.JSON(http.StatusInternalServerError, models.SpeechToTextError{
					Error:   "Internal server error",
					Message: "Failed to create temporary directory",
					Code:    "TEMP_DIR_ERROR",
				})
				return
			}

			// Generate unique filename to prevent collisions
			uniqueFileName := generateUniqueFileName(file.Filename)
			tempFilePath := fmt.Sprintf("%s/%s", tempDir, uniqueFileName)
			if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
				logger.Error("Failed to save uploaded file", "error", err)
				c.JSON(http.StatusInternalServerError, models.SpeechToTextError{
					Error:   "Internal server error",
					Message: "Failed to save uploaded file",
					Code:    "FILE_SAVE_ERROR",
				})
				return
			}

			// Clean up temporary file after processing
			defer func() {
				if err := os.Remove(tempFilePath); err != nil {
					logger.Error("Failed to remove temporary file", "error", err, "file", tempFilePath)
				}
			}()

			// Detect sample rate
			sampleRate, err := speechToTextService.DetectSampleRate(tempFilePath)
			if err != nil {
				logger.Error("Failed to detect sample rate", "error", err)
				// Continue with default sample rate
				sampleRate = 16000
			}

			// Transcribe audio to text
			ctx := context.Background()
			transcript, err := speechToTextService.TranscribeAudio(ctx, tempFilePath, languageCode)
			if err != nil {
				logger.Error("Speech-to-text transcription failed", "error", err)
				c.JSON(http.StatusInternalServerError, models.SpeechToTextError{
					Error:   "Transcription failed",
					Message: "Unable to transcribe audio. Please try again.",
					Code:    "TRANSCRIPTION_FAILED",
				})
				return
			}

			// Get target languages for translation (all except the original)
			allLanguages := []string{"en-IN", "hi-IN", "mr-IN", "gu-IN"}
			var targetLanguages []string
			for _, lang := range allLanguages {
				if lang != languageCode {
					targetLanguages = append(targetLanguages, lang)
				}
			}

			// Translate transcript to other languages
			translations, err := translationService.TranslateToMultipleLanguagesConcurrent(ctx, transcript, languageCode, targetLanguages)
			if err != nil {
				logger.Error("Translation failed", "error", err)
				// Return partial results if translation fails
				translations = make(map[string]string)
			}

			// Apply final train name correction to original transcript
			correctedTranscript := applyTrainNameCorrection(transcript, languageCode)

			// Apply final train name correction to all translations
			correctedTranslations := make(map[string]string)
			for lang, translation := range translations {
				correctedTranslations[lang] = applyTrainNameCorrection(translation, lang)
			}

			// Return response
			response := models.SpeechToTextResponse{
				OriginalTranscript: correctedTranscript,
				OriginalLanguage:   languageCode,
				SampleRate:         sampleRate,
				AudioFormat:        contentType,
				FileSize:           file.Size,
				Translations:       correctedTranslations,
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
				"health":                "/api/v1/health",
				"text-translate":        "/api/v1/text-translate",
				"audio-language-detect": "/api/v1/audio-language-detect",
				"speech-to-text":        "/api/v1/speech-to-text",
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
